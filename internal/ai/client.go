package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Mattel-Limbo/larasense-limbo/internal/config"
)

const (
	defaultTimeout    = 120 * time.Second
	maxRetries        = 3
	retryBaseDelay    = 2 * time.Second
)

// Client handles communication with the AI provider API.
type Client struct {
	cfg        *config.ProviderConfig
	httpClient *http.Client
	verbose    bool
}

// NewClient creates a new AI provider client.
func NewClient(cfg *config.ProviderConfig, verbose bool) *Client {
	return &Client{
		cfg:     cfg,
		verbose: verbose,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// chatRequest is the payload for /v1/chat/completions endpoint.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

// chatMessage represents a single message in the chat format.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// responsesRequest is the payload for /v1/responses endpoint.
type responsesRequest struct {
	Model        string `json:"model"`
	Instructions string `json:"instructions"`
	Input        string `json:"input"`
}

// Response is the parsed response from the AI provider.
type Response struct {
	Issues []Issue `json:"issues"`
}

// Issue represents a single code review finding.
type Issue struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Severity    string `json:"severity"`
	Suggestion  string `json:"suggestion"`
}

// Analyze sends the diff and context to the AI provider and returns structured issues.
func (c *Client) Analyze(diffText, contextText string) (*Response, error) {
	systemPrompt := buildPrompt()
	userContent := fmt.Sprintf("## File Context\n\n%s\n\n## Git Diff\n\n%s", contextText, diffText)

	var jsonBody []byte
	var err error

	endpoint := strings.ToLower(c.cfg.Endpoint)
	if endpoint == "responses" {
		reqBody := responsesRequest{
			Model:        c.cfg.Model,
			Instructions: systemPrompt,
			Input:        userContent,
		}
		jsonBody, err = json.Marshal(reqBody)
	} else {
		// Default: chat completions format (most compatible)
		reqBody := chatRequest{
			Model: c.cfg.Model,
			Messages: []chatMessage{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: userContent},
			},
		}
		jsonBody, err = json.Marshal(reqBody)
	}

	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	// Retry loop with exponential backoff
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryBaseDelay * time.Duration(1<<(attempt-1))
			time.Sleep(delay)
		}

		resp, err := c.doRequest(jsonBody)
		if err != nil {
			lastErr = err
			continue
		}
		return resp, nil
	}

	return nil, fmt.Errorf("AI request failed after %d attempts: %w", maxRetries, lastErr)
}

// doRequest performs a single HTTP request to the AI provider.
func (c *Client) doRequest(body []byte) (*Response, error) {
	endpoint := strings.ToLower(c.cfg.Endpoint)
	path := "/v1/chat/completions"
	if endpoint == "responses" {
		path = "/v1/responses"
	}
	url := strings.TrimRight(c.cfg.BaseURL, "/") + path

	if c.verbose {
		log.Printf("[VERBOSE] POST %s", url)
		log.Printf("[VERBOSE] Model: %s", c.cfg.Model)
		log.Printf("[VERBOSE] Request body (%d bytes):\n%s", len(body), truncate(string(body), 2000))
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		if c.verbose {
			log.Printf("[VERBOSE] Request failed after %s: %v", elapsed, err)
		}
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if c.verbose {
		log.Printf("[VERBOSE] Response status: %d (%s)", resp.StatusCode, elapsed)
		log.Printf("[VERBOSE] Response body (%d bytes):\n%s", len(respBody), truncate(string(respBody), 2000))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI provider returned status %d: %s", resp.StatusCode, truncate(string(respBody), 500))
	}

	// Parse the AI provider response.
	// The response structure varies by provider. We attempt to extract the
	// text content and then parse the JSON issues from it.
	aiResp, err := extractContent(respBody)
	if err != nil {
		return nil, fmt.Errorf("parsing AI response: %w", err)
	}

	if c.verbose {
		log.Printf("[VERBOSE] Parsed %d issue(s) from response", len(aiResp.Issues))
	}

	return aiResp, nil
}

// extractContent parses the AI provider's response and extracts the review issues.
// It handles the common OpenAI-compatible response format.
func extractContent(body []byte) (*Response, error) {
	// Try OpenAI Responses API format first
	var responsesAPI struct {
		Output []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(body, &responsesAPI); err == nil && len(responsesAPI.Output) > 0 {
		for _, out := range responsesAPI.Output {
			for _, c := range out.Content {
				if c.Text != "" {
					return parseIssuesFromText(c.Text)
				}
			}
		}
	}

	// Try OpenAI Chat Completions format
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &chatResp); err == nil && len(chatResp.Choices) > 0 {
		content := chatResp.Choices[0].Message.Content
		if content != "" {
			return parseIssuesFromText(content)
		}
	}

	// Try direct JSON response (the body itself is the issues JSON)
	var directResp Response
	if err := json.Unmarshal(body, &directResp); err == nil && len(directResp.Issues) > 0 {
		return &directResp, nil
	}

	return nil, fmt.Errorf("could not extract content from AI response: %s", truncate(string(body), 300))
}

// parseIssuesFromText extracts JSON issues from a text response that may contain
// markdown code fences or other wrapping.
func parseIssuesFromText(text string) (*Response, error) {
	// Strip markdown code fences if present
	cleaned := text
	cleaned = strings.TrimSpace(cleaned)

	// Remove ```json ... ``` wrapping
	if strings.HasPrefix(cleaned, "```") {
		lines := strings.Split(cleaned, "\n")
		// Remove first line (```json) and last line (```)
		if len(lines) > 2 {
			start := 1
			end := len(lines) - 1
			if strings.TrimSpace(lines[end]) == "```" || strings.TrimSpace(lines[end]) == "" {
				// Find the closing ```
				for i := len(lines) - 1; i >= 0; i-- {
					if strings.TrimSpace(lines[i]) == "```" {
						end = i
						break
					}
				}
			}
			cleaned = strings.Join(lines[start:end], "\n")
		}
	}

	cleaned = strings.TrimSpace(cleaned)

	var resp Response
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		// Try to find JSON object in the text
		startIdx := strings.Index(cleaned, "{")
		endIdx := strings.LastIndex(cleaned, "}")
		if startIdx >= 0 && endIdx > startIdx {
			jsonStr := cleaned[startIdx : endIdx+1]
			if err2 := json.Unmarshal([]byte(jsonStr), &resp); err2 != nil {
				return nil, fmt.Errorf("parsing issues JSON: %w (raw: %s)", err2, truncate(cleaned, 200))
			}
			return &resp, nil
		}
		return nil, fmt.Errorf("parsing issues JSON: %w (raw: %s)", err, truncate(cleaned, 200))
	}

	return &resp, nil
}

// buildPrompt returns the system prompt for the AI code reviewer.
func buildPrompt() string {
	return `You are a Senior Laravel Developer performing a strict code review.

Analyze the provided git diff and file context. Focus ONLY on:

1. **Performance Issues**: N+1 queries, unnecessary database queries, missing eager loading, inefficient loops
2. **Security Issues**: Missing validation, mass assignment vulnerabilities, SQL injection risks, XSS in Blade templates, CSRF issues
3. **Bad Practices**: Fat controllers, business logic in Blade views, missing form requests, improper error handling, hardcoded values
4. **Laravel Conventions**: Improper use of Eloquent, missing route model binding, incorrect naming conventions, missing middleware

Rules:
- Only report issues found in the CHANGED lines (the diff), not in surrounding context
- Be specific about the file and line number
- Provide actionable suggestions
- Rate severity as: low, medium, or high
- Do NOT report style-only issues (formatting, spacing)
- Do NOT report issues in test files

You MUST respond with ONLY valid JSON in this exact format (no markdown, no explanation, just JSON):

{
  "issues": [
    {
      "title": "Brief issue title",
      "description": "Detailed explanation of the problem",
      "file": "path/to/file.php",
      "line": 42,
      "severity": "low|medium|high",
      "suggestion": "How to fix this issue"
    }
  ]
}

If there are no issues, respond with: {"issues": []}`
}

// truncate shortens a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

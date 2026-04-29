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

// responseFormat enforces JSON output from the AI provider.
type responseFormat struct {
	Type string `json:"type"`
}

// chatRequest is the payload for /v1/chat/completions endpoint.
type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Temperature    *float64        `json:"temperature,omitempty"`
	Seed           *int            `json:"seed,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

// chatMessage represents a single message in the chat format.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// responsesRequest is the payload for /v1/responses endpoint.
type responsesRequest struct {
	Model        string   `json:"model"`
	Instructions string   `json:"instructions"`
	Input        string   `json:"input"`
	MaxTokens    int      `json:"max_output_tokens,omitempty"`
	Temperature  *float64 `json:"temperature,omitempty"`
	Seed         *int     `json:"seed,omitempty"`
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

// Analyze sends the diff and context to the AI provider using the diff review prompt.
func (c *Client) Analyze(diffText, contextText, customPrompt string) (*Response, error) {
	systemPrompt := BuildDiffPrompt()
	if customPrompt != "" {
		systemPrompt += "\n\nAdditional instructions from the user:\n" + customPrompt
	}
	userContent := fmt.Sprintf("## File Context\n\n%s\n\n## Git Diff\n\n%s", contextText, diffText)
	return c.send(systemPrompt, userContent)
}

// AnalyzeWithPrompt sends content to the AI provider using a caller-provided system prompt.
func (c *Client) AnalyzeWithPrompt(systemPrompt, userContent string) (*Response, error) {
	return c.send(systemPrompt, userContent)
}

func (c *Client) send(systemPrompt, userContent string) (*Response, error) {

	var jsonBody []byte
	var err error

	// Prepare optional parameters
	var tempPtr *float64
	if c.cfg.Temperature >= 0 {
		temp := c.cfg.Temperature
		tempPtr = &temp
	}

	var seedPtr *int
	if c.cfg.Seed > 0 {
		seed := c.cfg.Seed
		seedPtr = &seed
	}

	// Calculate effective max_tokens: use configured value, but cap based on
	// estimated prompt size to avoid wasting budget on overly verbose responses.
	effectiveMaxTokens := c.cfg.MaxTokens
	if effectiveMaxTokens > 0 {
		// Estimate prompt tokens (~4 chars per token)
		promptChars := len(systemPrompt) + len(userContent)
		estimatedPromptTokens := promptChars / 4

		// If prompt is large, reduce completion budget proportionally
		// to encourage concise output within the user's max_tokens limit
		if estimatedPromptTokens > 2000 && effectiveMaxTokens > 512 {
			// Scale down: large prompts need less verbose responses
			scaledMax := effectiveMaxTokens * 2000 / estimatedPromptTokens
			if scaledMax < 512 {
				scaledMax = 512 // minimum floor
			}
			if scaledMax < effectiveMaxTokens {
				effectiveMaxTokens = scaledMax
				if c.verbose {
					log.Printf("[VERBOSE] Scaled max_tokens from %d to %d (prompt ~%d tokens)",
						c.cfg.MaxTokens, effectiveMaxTokens, estimatedPromptTokens)
				}
			}
		}
	}

	endpoint := strings.ToLower(c.cfg.Endpoint)
	if endpoint == "responses" {
		reqBody := responsesRequest{
			Model:        c.cfg.Model,
			Instructions: systemPrompt,
			Input:        userContent,
			MaxTokens:    effectiveMaxTokens,
			Temperature:  tempPtr,
			Seed:         seedPtr,
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
			MaxTokens:      effectiveMaxTokens,
			Temperature:    tempPtr,
			Seed:           seedPtr,
			ResponseFormat: &responseFormat{Type: "json_object"},
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
// markdown code fences, surrounding text, or full markdown responses.
func parseIssuesFromText(text string) (*Response, error) {
	cleaned := strings.TrimSpace(text)

	// Try 1: Direct JSON parse
	var resp Response
	if err := json.Unmarshal([]byte(cleaned), &resp); err == nil {
		return &resp, nil
	}

	// Try 2: Extract from markdown code fences (```json ... ```)
	if extracted := extractFromCodeFence(cleaned); extracted != "" {
		if err := json.Unmarshal([]byte(extracted), &resp); err == nil {
			return &resp, nil
		}
	}

	// Try 3: Find {"issues" pattern anywhere in the text
	if idx := strings.Index(cleaned, `{"issues"`); idx >= 0 {
		candidate := cleaned[idx:]
		endIdx := findMatchingBrace(candidate)
		if endIdx > 0 {
			jsonStr := candidate[:endIdx+1]
			if err := json.Unmarshal([]byte(jsonStr), &resp); err == nil {
				return &resp, nil
			}
		}
	}

	// Try 4: Find any JSON object with braces
	startIdx := strings.Index(cleaned, "{")
	endIdx := strings.LastIndex(cleaned, "}")
	if startIdx >= 0 && endIdx > startIdx {
		jsonStr := cleaned[startIdx : endIdx+1]
		if err := json.Unmarshal([]byte(jsonStr), &resp); err == nil {
			return &resp, nil
		}
	}

	// Try 5: Parse Markdown response as fallback (some models ignore JSON-only instruction)
	if issues := parseMarkdownFallback(cleaned); len(issues) > 0 {
		return &Response{Issues: issues}, nil
	}

	return nil, fmt.Errorf("parsing issues JSON: no valid JSON found in AI response (raw: %s)", truncate(cleaned, 200))
}

func extractFromCodeFence(text string) string {
	lines := strings.Split(text, "\n")
	inFence := false
	var jsonLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inFence && (trimmed == "```json" || trimmed == "```") {
			inFence = true
			continue
		}
		if inFence && trimmed == "```" {
			break
		}
		if inFence {
			jsonLines = append(jsonLines, line)
		}
	}

	if len(jsonLines) > 0 {
		return strings.TrimSpace(strings.Join(jsonLines, "\n"))
	}
	return ""
}

// findMatchingBrace finds the index of the closing brace that matches the opening brace at index 0.
func findMatchingBrace(s string) int {
	depth := 0
	inString := false
	escaped := false

	for i, ch := range s {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '{' {
			depth++
		}
		if ch == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// parseMarkdownFallback extracts issues from a Markdown-formatted AI response.
// This handles models that ignore JSON-only instructions and return Markdown instead.
func parseMarkdownFallback(text string) []Issue {
	var issues []Issue
	lines := strings.Split(text, "\n")

	var currentFile string
	var currentTitle string
	var currentDesc strings.Builder
	var currentSeverity string
	var currentLine int
	inIssue := false

	flushIssue := func() {
		if currentTitle != "" {
			desc := strings.TrimSpace(currentDesc.String())
			if desc == "" {
				desc = currentTitle
			}
			issues = append(issues, Issue{
				Title:       currentTitle,
				Description: desc,
				File:        currentFile,
				Line:        currentLine,
				Severity:    currentSeverity,
				Suggestion:  "",
			})
		}
		currentTitle = ""
		currentDesc.Reset()
		currentSeverity = "medium"
		currentLine = 0
		inIssue = false
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect file headers: "## File: `path/to/file.php`"
		if strings.HasPrefix(trimmed, "## File:") || strings.HasPrefix(trimmed, "## File ") {
			flushIssue()
			currentFile = extractBacktickContent(trimmed)
			continue
		}

		// Detect issue headers: "### Title" or "### Bug: Title"
		if strings.HasPrefix(trimmed, "### ") {
			flushIssue()
			currentTitle = strings.TrimPrefix(trimmed, "### ")
			currentSeverity = detectSeverityFromText(currentTitle)
			inIssue = true
			continue
		}

		// Detect severity markers in body
		if inIssue {
			lower := strings.ToLower(trimmed)
			if strings.Contains(lower, "critical") || strings.Contains(lower, "\xf0\x9f\x94\xb4") {
				currentSeverity = "high"
			} else if strings.Contains(lower, "\xf0\x9f\x9f\xa1") && currentSeverity != "high" {
				currentSeverity = "medium"
			} else if strings.Contains(lower, "\xf0\x9f\x9f\xa2") && currentSeverity == "medium" {
				currentSeverity = "low"
			}

			// Detect line references: "**Lines affected:** 60, 65" or "**Line 42**"
			if currentLine == 0 && (strings.Contains(lower, "line") || strings.Contains(lower, "lines")) {
				if n := extractFirstNumber(trimmed); n > 0 {
					currentLine = n
				}
			}

			if trimmed != "" && !strings.HasPrefix(trimmed, "---") && !strings.HasPrefix(trimmed, "```") && !strings.HasPrefix(trimmed, "|") {
				currentDesc.WriteString(trimmed)
				currentDesc.WriteString(" ")
			}
		}
	}
	flushIssue()

	return issues
}

func extractBacktickContent(s string) string {
	start := strings.Index(s, "`")
	if start < 0 {
		return ""
	}
	end := strings.Index(s[start+1:], "`")
	if end < 0 {
		return ""
	}
	return s[start+1 : start+1+end]
}

func detectSeverityFromText(title string) string {
	lower := strings.ToLower(title)
	if strings.Contains(lower, "critical") || strings.Contains(lower, "bug") || strings.Contains(lower, "security") || strings.Contains(lower, "vulnerability") {
		return "high"
	}
	if strings.Contains(lower, "missing") || strings.Contains(lower, "unbounded") || strings.Contains(lower, "duplicate") {
		return "medium"
	}
	return "medium"
}

func extractFirstNumber(s string) int {
	num := 0
	inNum := false
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			num = num*10 + int(ch-'0')
			inNum = true
		} else if inNum {
			break
		}
	}
	return num
}

// BuildDiffPrompt returns the system prompt for diff-based code review.
// Optimized for minimal token usage, JSON compliance, and consistent results.
func BuildDiffPrompt() string {
	return `You are a JSON-only API. You output raw JSON with no markdown, no explanation, no wrapping.

Review the Laravel git diff. For each file, check this exact checklist in order:
1. SECURITY: mass assignment, SQL injection, XSS, CSRF, missing validation, hardcoded secrets
2. PERFORMANCE: N+1 queries, missing eager loading, unbounded queries, inefficient loops
3. BUGS: null safety, undefined variables, type errors, race conditions
4. BAD PRACTICES: fat controllers, logic in views, missing Form Requests, tight coupling
5. CONVENTIONS: Eloquent misuse, missing route model binding, missing middleware

Rules: only CHANGED lines, specific file+line, severity low/medium/high, skip style-only issues, skip test files. Each description and suggestion must be exactly 1 sentence.

Output ONLY this JSON structure:
{"issues":[{"title":"string","description":"string","file":"string","line":0,"severity":"low|medium|high","suggestion":"string"}]}

No issues found: {"issues":[]}`
}

// BuildScanPrompt returns the system prompt for full codebase scan.
// Optimized for minimal token usage, JSON compliance, and consistent results.
func BuildScanPrompt() string {
	return `You are a JSON-only API. You output raw JSON with no markdown, no explanation, no wrapping.

Audit the Laravel source files. For each file, check this exact checklist in order:
1. SECURITY: mass assignment, SQL injection, XSS, CSRF, missing validation, hardcoded secrets
2. PERFORMANCE: N+1 queries, missing eager loading, unbounded queries, inefficient loops
3. BUGS: null safety, undefined variables, type errors, race conditions
4. BAD PRACTICES: fat controllers, logic in views, missing Form Requests, tight coupling
5. CONVENTIONS: Eloquent misuse, missing route model binding, missing middleware

Rules: review entire file, specific file+line, severity low/medium/high, skip style-only issues, skip test files, prioritize high-impact issues. Each description and suggestion must be exactly 1 sentence.

Output ONLY this JSON structure:
{"issues":[{"title":"string","description":"string","file":"string","line":0,"severity":"low|medium|high","suggestion":"string"}]}

No issues found: {"issues":[]}`
}

// truncate shortens a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

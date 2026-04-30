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
	maxTokensCap      = 65536
)

type truncatedError struct {
	partialResponse *Response
	message         string
}

func (e *truncatedError) Error() string { return e.message }

// Client handles communication with the AI provider API.
type Client struct {
	cfg            *config.ProviderConfig
	httpClient     *http.Client
	verbose        bool
	lastUserContent string // stored for markdown fallback file inference
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
	Fix         *Fix   `json:"fix,omitempty"`
}

type Fix struct {
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Before    string `json:"before"`
	After     string `json:"after"`
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
	c.lastUserContent = userContent

	var jsonBody []byte
	var err error

	// Always send temperature for deterministic output (0.0 = fully deterministic)
	temp := c.cfg.Temperature
	tempPtr := &temp

	var seedPtr *int
	if c.cfg.Seed > 0 {
		seed := c.cfg.Seed
		seedPtr = &seed
	}

	// Use configured max_tokens directly — no scaling.
	// The user controls this via config. Default: 4096.
	effectiveMaxTokens := c.cfg.MaxTokens

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

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryBaseDelay * time.Duration(1<<(attempt-1))
			time.Sleep(delay)
		}

		resp, err := c.doRequest(jsonBody)
		if err != nil {
			// On truncation: retry once with 2x max_tokens
			if te, ok := err.(*truncatedError); ok {
				doubled := effectiveMaxTokens * 2
				if doubled > maxTokensCap {
					doubled = maxTokensCap
				}
				if doubled > effectiveMaxTokens {
					log.Printf("[retry] Response truncated — retrying with max_tokens %d → %d", effectiveMaxTokens, doubled)
					jsonBody = c.rebuildRequestBody(systemPrompt, userContent, tempPtr, seedPtr, doubled, endpoint)
					effectiveMaxTokens = doubled
					lastErr = te
					continue
				}
			}
			lastErr = err
			continue
		}
		return resp, nil
	}

	return nil, fmt.Errorf("AI request failed after %d attempts: %w", maxRetries, lastErr)
}

func (c *Client) rebuildRequestBody(systemPrompt, userContent string, temp *float64, seed *int, maxTokens int, endpoint string) []byte {
	var jsonBody []byte
	if endpoint == "responses" {
		reqBody := responsesRequest{
			Model:        c.cfg.Model,
			Instructions: systemPrompt,
			Input:        userContent,
			MaxTokens:    maxTokens,
			Temperature:  temp,
			Seed:         seed,
		}
		jsonBody, _ = json.Marshal(reqBody)
	} else {
		reqBody := chatRequest{
			Model: c.cfg.Model,
			Messages: []chatMessage{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: userContent},
			},
			MaxTokens:      maxTokens,
			Temperature:    temp,
			Seed:           seed,
			ResponseFormat: &responseFormat{Type: "json_object"},
		}
		jsonBody, _ = json.Marshal(reqBody)
	}
	return jsonBody
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
	aiResp, err := extractContent(respBody, c.lastUserContent)
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
func extractContent(body []byte, userContent ...string) (*Response, error) {
	// Try OpenAI Responses API format first
	var responsesAPI struct {
		Output []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	// Extract raw text content from response envelope
	var rawText string

	if err := json.Unmarshal(body, &responsesAPI); err == nil && len(responsesAPI.Output) > 0 {
		for _, out := range responsesAPI.Output {
			for _, c := range out.Content {
				if c.Text != "" {
					rawText = c.Text
					break
				}
			}
			if rawText != "" {
				break
			}
		}
	}

	var truncated bool

	if rawText == "" {
		var chatResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(body, &chatResp); err == nil && len(chatResp.Choices) > 0 {
			rawText = chatResp.Choices[0].Message.Content
			truncated = chatResp.Choices[0].FinishReason == "length"
		}
	}

	// Try parsing as JSON first
	if rawText != "" {
		resp, err := parseIssuesFromText(rawText)
		if err == nil {
			return resp, nil
		}

			if truncated {
			if repaired := repairTruncatedJSON(rawText); repaired != "" {
				resp, err = parseIssuesFromText(repaired)
				if err == nil {
					log.Printf("[warning] AI response was truncated — parsed %d issue(s) from partial response", len(resp.Issues))
					return resp, nil
				}
			}
			return nil, &truncatedError{
				message: fmt.Sprintf("AI response truncated (finish_reason: length) — increase max_tokens (raw: %s)", truncate(rawText, 200)),
			}
		}
	}

	// Try direct JSON response (the body itself is the issues JSON)
	var directResp Response
	if err := json.Unmarshal(body, &directResp); err == nil && len(directResp.Issues) > 0 {
		return &directResp, nil
	}

	// Last resort: parse as Markdown with file hints from user content
	if rawText != "" {
		var hints map[string]string
		if len(userContent) > 0 && userContent[0] != "" {
			hints = extractFileHints(userContent[0])
		}
		if issues := parseMarkdownFallback(rawText, hints); len(issues) > 0 {
			return &Response{Issues: issues}, nil
		}
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
	if issues := parseMarkdownFallback(cleaned, nil); len(issues) > 0 {
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
// fileHints maps section names to file paths (e.g., "Auditable" → "app/Traits/Auditable.php").
func parseMarkdownFallback(text string, fileHints map[string]string) []Issue {
	var issues []Issue
	lines := strings.Split(text, "\n")

	var currentFile string
	var currentTitle string
	var currentDesc strings.Builder
	var currentSuggestion strings.Builder
	var currentSeverity string
	var currentLine int
	var currentBeforeLines []string
	var currentAfterLines []string
	inIssue := false
	inCodeBlock := false
	codeBlockType := "" // "before" or "after"
	inProblem := false
	inFix := false

	flushIssue := func() {
		if currentTitle != "" {
			desc := strings.TrimSpace(currentDesc.String())
			suggestion := strings.TrimSpace(currentSuggestion.String())
			if desc == "" {
				desc = currentTitle
			}

			issue := Issue{
				Title:       currentTitle,
				Description: desc,
				File:        currentFile,
				Line:        currentLine,
				Severity:    currentSeverity,
				Suggestion:  suggestion,
			}

			// Attach fix if we have before/after code
			if len(currentBeforeLines) > 0 && len(currentAfterLines) > 0 && currentLine > 0 {
				before := strings.Join(currentBeforeLines, "\n")
				after := strings.Join(currentAfterLines, "\n")
				beforeCount := len(currentBeforeLines)
				issue.Fix = &Fix{
					StartLine: currentLine,
					EndLine:   currentLine + beforeCount - 1,
					Before:    before,
					After:     after,
				}
			}

			issues = append(issues, issue)
		}
		currentTitle = ""
		currentDesc.Reset()
		currentSuggestion.Reset()
		currentSeverity = "medium"
		currentLine = 0
		currentBeforeLines = nil
		currentAfterLines = nil
		inIssue = false
		inProblem = false
		inFix = false
		codeBlockType = ""
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle code blocks (``` ... ```)
		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				inCodeBlock = false
				continue
			}
			inCodeBlock = true
			// First code block after problem = before, after fix marker = after
			if inFix && len(currentBeforeLines) > 0 {
				codeBlockType = "after"
			} else if inIssue {
				codeBlockType = "before"
			}
			continue
		}

		if inCodeBlock {
			switch codeBlockType {
			case "before":
				currentBeforeLines = append(currentBeforeLines, line)
			case "after":
				currentAfterLines = append(currentAfterLines, line)
			}
			continue
		}

		// Detect file headers: "## File: `path`" or "## `path`" or "## Auditable Trait"
		if strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "### ") {
			flushIssue()
			if extracted := extractBacktickContent(trimmed); extracted != "" {
				currentFile = extracted
			} else {
				sectionName := strings.TrimPrefix(trimmed, "## ")
				sectionName = strings.TrimSpace(sectionName)
				// Try fileHints first, then infer from text
				if fileHints != nil {
					for key, path := range fileHints {
						if strings.Contains(sectionName, key) {
							currentFile = path
							break
						}
					}
				}
				if currentFile == "" {
					if inferred := inferFileFromSection(sectionName, text); inferred != "" {
						currentFile = inferred
					}
				}
			}
			continue
		}

		// Detect issue headers: "### Title" or "#### 1. Title" or "#### Title"
		if strings.HasPrefix(trimmed, "#### ") || (strings.HasPrefix(trimmed, "### ") && !strings.Contains(strings.ToLower(trimmed), "issues found")) {
			flushIssue()
			header := trimmed
			if strings.HasPrefix(header, "#### ") {
				header = strings.TrimPrefix(header, "#### ")
			} else {
				header = strings.TrimPrefix(header, "### ")
			}
			// Remove leading number: "1. **Critical: Title**" → "Critical: Title"
			if idx := strings.Index(header, ". "); idx >= 0 && idx <= 3 {
				header = header[idx+2:]
			}
			// Remove bold markers
			header = strings.ReplaceAll(header, "**", "")
			// Remove trailing line reference: "(Line ~108-110)"
			if parenIdx := strings.LastIndex(header, "(Line"); parenIdx > 0 {
				lineRef := header[parenIdx:]
				header = strings.TrimSpace(header[:parenIdx])
				if n := extractFirstNumber(lineRef); n > 0 {
					currentLine = n
				}
			}
			currentTitle = strings.TrimSpace(header)
			currentSeverity = detectSeverityFromText(currentTitle + " " + trimmed)
			inIssue = true
			inProblem = false
			inFix = false
			currentBeforeLines = nil
			currentAfterLines = nil
			continue
		}

		// Skip "### Issues Found" group headers
		if strings.HasPrefix(trimmed, "### ") && strings.Contains(strings.ToLower(trimmed), "issues found") {
			continue
		}

		if inIssue {
			lower := strings.ToLower(trimmed)

			// Detect **Problem:** / **Fix:** markers
			if strings.Contains(trimmed, "**Problem:**") || strings.Contains(trimmed, "**Problem**") {
				inProblem = true
				inFix = false
				// Extract inline problem text after marker
				if idx := strings.Index(trimmed, ":**"); idx > 0 {
					rest := strings.TrimSpace(trimmed[idx+3:])
					if rest != "" {
						currentDesc.WriteString(rest)
						currentDesc.WriteString(" ")
					}
				}
				continue
			}
			if strings.Contains(trimmed, "**Fix:**") || strings.Contains(trimmed, "**Fix**") || strings.HasPrefix(trimmed, "**Fix") {
				inProblem = false
				inFix = true
				// Extract inline fix text after marker
				if idx := strings.Index(trimmed, ":**"); idx > 0 {
					rest := strings.TrimSpace(trimmed[idx+3:])
					if rest != "" {
						currentSuggestion.WriteString(rest)
						currentSuggestion.WriteString(" ")
					}
				}
				continue
			}
			if strings.HasPrefix(trimmed, "**Severity:") {
				if strings.Contains(lower, "high") || strings.Contains(lower, "critical") {
					currentSeverity = "high"
				} else if strings.Contains(lower, "medium") {
					currentSeverity = "medium"
				} else if strings.Contains(lower, "low") {
					currentSeverity = "low"
				}
				continue
			}

			// Detect severity from text
			if strings.Contains(lower, "critical") {
				currentSeverity = "high"
			}

			// Detect line references
			if currentLine == 0 && (strings.Contains(lower, "line") || strings.Contains(lower, "lines")) {
				if n := extractFirstNumber(trimmed); n > 0 {
					currentLine = n
				}
			}

			// Accumulate description/suggestion text
			if trimmed != "" && !strings.HasPrefix(trimmed, "---") && !strings.HasPrefix(trimmed, "|") {
				if inFix && !inProblem {
					currentSuggestion.WriteString(trimmed)
					currentSuggestion.WriteString(" ")
				} else {
					currentDesc.WriteString(trimmed)
					currentDesc.WriteString(" ")
				}
			}
		}
	}
	flushIssue()

	return issues
}

// repairTruncatedJSON attempts to fix a JSON response that was cut off mid-stream.
// It finds the last complete issue object and closes the JSON structure.
func repairTruncatedJSON(text string) string {
	text = strings.TrimSpace(text)

	// Must start with {"issues":[ to be repairable
	if !strings.HasPrefix(text, `{"issues":[`) {
		return ""
	}

	// Strategy: find the last complete issue object by looking for the last "},"
	// or the last "}" that closes an issue object
	lastCompleteComma := strings.LastIndex(text, "},")
	lastCompleteBrace := strings.LastIndex(text, `}]`)

	cutPoint := -1
	if lastCompleteBrace > lastCompleteComma && lastCompleteBrace > 0 {
		// Already has a complete array — just needs closing
		cutPoint = lastCompleteBrace + 2
	} else if lastCompleteComma > 0 {
		// Has at least one complete issue followed by comma — cut after it and close
		cutPoint = lastCompleteComma + 1
	}

	if cutPoint <= 0 {
		return ""
	}

	repaired := text[:cutPoint] + "]}"

	// Verify it's valid JSON
	var resp Response
	if err := json.Unmarshal([]byte(repaired), &resp); err != nil {
		return ""
	}

	return repaired
}

// extractFileHints parses user content to build a map of class/trait names to file paths.
// Input format: "File: app/Traits/Auditable.php [php]\n..."
// Output: {"Auditable": "app/Traits/Auditable.php", "HandleUploadedFile": "app/Traits/HandleUploadedFile.php"}
func extractFileHints(userContent string) map[string]string {
	hints := make(map[string]string)
	for _, line := range strings.Split(userContent, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "File:") {
			continue
		}
		// "File: app/Traits/Auditable.php [php]" → "app/Traits/Auditable.php"
		rest := strings.TrimPrefix(trimmed, "File:")
		rest = strings.TrimSpace(rest)
		parts := strings.Fields(rest)
		if len(parts) == 0 {
			continue
		}
		filePath := parts[0]
		if !strings.HasSuffix(filePath, ".php") {
			continue
		}

		// Extract class name from path: "app/Traits/Auditable.php" → "Auditable"
		base := filePath
		if lastSlash := strings.LastIndex(base, "/"); lastSlash >= 0 {
			base = base[lastSlash+1:]
		}
		className := strings.TrimSuffix(base, ".php")
		if className != "" {
			hints[className] = filePath
		}
	}
	return hints
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

// inferFileFromSection tries to match a section name like "Auditable Trait" or
// "HandleUploadedFile" to a file path. Uses multiple strategies:
// 1. Search for "File: path" pattern in the full text
// 2. Search for "ClassName::method" patterns that reference the section name
// 3. Map common Laravel names to paths
func inferFileFromSection(sectionName, fullText string) string {
	// Clean section name: "Auditable Trait" → "Auditable"
	name := sectionName
	// Remove markdown formatting
	name = strings.ReplaceAll(name, "`", "")
	name = strings.ReplaceAll(name, "**", "")
	// Remove type suffixes
	for _, suffix := range []string{" Trait", " Traits", " Controller", " Model", " Service", " Class", " Helper"} {
		name = strings.TrimSuffix(name, suffix)
	}
	// Remove "Code Review:" prefix
	if idx := strings.Index(name, ":"); idx >= 0 {
		candidate := strings.TrimSpace(name[idx+1:])
		if candidate != "" {
			name = candidate
		}
	}
	// Remove "and" joined names — take first: "Auditable and HandleUploadedFile" → "Auditable"
	if idx := strings.Index(name, " and "); idx > 0 {
		name = name[:idx]
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	// Strategy 1: Search for "File: .../<name>.php" in the full text
	for _, line := range strings.Split(fullText, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "File:") && strings.Contains(trimmed, name) {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				filePath := parts[1]
				// Remove trailing brackets like "[php]"
				if bracketIdx := strings.Index(filePath, "["); bracketIdx > 0 {
					filePath = filePath[:bracketIdx]
				}
				filePath = strings.TrimSpace(filePath)
				if strings.HasSuffix(filePath, ".php") {
					return filePath
				}
			}
		}
	}

	return ""
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
// Designed for deterministic, consistent results across multiple runs.
func BuildDiffPrompt() string {
	return `You are a deterministic code scanner. Output raw JSON only. No markdown. No explanation.

Scan the Laravel git diff for ONLY these specific patterns. Report a match ONLY if the exact pattern exists in CHANGED lines.

SEVERITY IS FIXED — do not reassign:

HIGH (report if found):
- SQL injection: raw DB queries with user input without parameterized binding
- Mass assignment: $request->all() or unguarded fill() without $fillable/$guarded
- XSS: {!! !!} or unescaped output with user-controlled data in Blade
- Hardcoded secrets: API keys, passwords, tokens as string literals
- Missing auth: public routes/controllers handling sensitive data without middleware
- Null reference: calling methods on potentially null values (e.g. request()->route()->getName() without null check)
- Undefined variable: using a variable that was never defined or assigned in current scope
- Wrong variable in loop: loop iterates as $x but body references $y (different variable name)

MEDIUM (report if found):
- N+1 query: DB query inside foreach/loop without eager loading
- Unbounded query: Model::all() or query without limit/pagination on large tables
- Fat controller: controller method >30 lines with business logic not in service/action class
- Missing validation: store/update without Form Request or validate()
- Double write: create() followed by immediate save() on same model
- Missing error handling: external calls (HTTP, file, queue) without try/catch
- Unreachable code: code after return/throw/exit that will never execute
- Wrong comparison: using = instead of == or === in conditions, or inverted logic (e.g. && vs ||)
- Type mismatch: passing wrong type to method (string where int expected, array where object expected)

LOW (report if found):
- Tight coupling: direct new ClassName() instead of dependency injection
- Missing route model binding: manual Model::find($id) in controller
- Naming violation: non-standard Laravel naming (controller not suffixed, model plural)
- Dead code: unused variables, unused imports, unreferenced private methods

RULES:
- Only report patterns found in CHANGED lines (+ lines in diff)
- Each issue: 1 sentence description, 1 sentence suggestion
- Do NOT invent issues. If no pattern matches, return empty.
- Do NOT report style/formatting issues

{"issues":[{"title":"string","description":"string","file":"string","line":0,"severity":"high|medium|low","suggestion":"string"}]}
Empty: {"issues":[]}`
}

// BuildScanPrompt returns the system prompt for full codebase scan.
// Designed for deterministic, consistent results across multiple runs.
func BuildScanPrompt() string {
	return `You are a deterministic code scanner. Output raw JSON only. No markdown. No explanation.

Scan the Laravel source files for ONLY these specific patterns. Report a match ONLY if the exact pattern exists.

SEVERITY IS FIXED — do not reassign:

HIGH (report if found):
- SQL injection: raw DB queries with user input without parameterized binding
- Mass assignment: $request->all() or unguarded fill() without $fillable/$guarded
- XSS: {!! !!} or unescaped output with user-controlled data in Blade
- Hardcoded secrets: API keys, passwords, tokens as string literals
- Missing auth: public routes/controllers handling sensitive data without middleware
- Null reference: calling methods on potentially null values without null check
- Undefined variable: using a variable that was never defined or assigned in current scope
- Wrong variable in loop: loop iterates as $x but body references $y (different variable name)

MEDIUM (report if found):
- N+1 query: DB query inside foreach/loop without eager loading
- Unbounded query: Model::all() or query without limit/pagination on large tables
- Fat controller: controller method >30 lines with business logic not in service/action class
- Missing validation: store/update without Form Request or validate()
- Double write: create() followed by immediate save() on same model
- Missing error handling: external calls (HTTP, file, queue) without try/catch
- Unreachable code: code after return/throw/exit that will never execute
- Wrong comparison: using = instead of == or === in conditions, or inverted logic (e.g. && vs ||)
- Type mismatch: passing wrong type to method (string where int expected, array where object expected)

LOW (report if found):
- Tight coupling: direct new ClassName() instead of dependency injection
- Missing route model binding: manual Model::find($id) in controller
- Naming violation: non-standard Laravel naming (controller not suffixed, model plural)
- Dead code: unused variables, unused imports, unreferenced private methods

RULES:
- Scan entire file content
- Each issue: 1 sentence description, 1 sentence suggestion
- Do NOT invent issues. If no pattern matches, return empty.
- Do NOT report style/formatting issues

{"issues":[{"title":"string","description":"string","file":"string","line":0,"severity":"high|medium|low","suggestion":"string"}]}
Empty: {"issues":[]}`
}

const fixPromptSuffix = `

ADDITIONAL: For each HIGH and MEDIUM severity issue, include a "fix" object with the exact code replacement.
- "start_line" and "end_line": the line range to replace (1-indexed, inclusive)
- "before": the EXACT original code from those lines (copy verbatim, preserve indentation)
- "after": the corrected replacement code (drop-in replacement)
- Do NOT include "fix" for LOW severity or issues requiring structural refactoring
- If you cannot provide an exact fix, omit the "fix" field for that issue
- Each issue MUST be a SEPARATE object in the array — do NOT combine multiple issues into one

CRITICAL: You MUST output ONLY a single raw JSON object. No markdown. No explanation. No code fences. Just the JSON.

Example of CORRECT output with 2 issues and fixes:
{"issues":[{"title":"Null reference on route()","description":"request()->route() can return null","file":"app/Traits/Auditable.php","line":108,"severity":"high","suggestion":"Add null check","fix":{"start_line":108,"end_line":110,"before":"            'url' => method_exists($this, 'replaceAuditRouteUrl')\n                ? $this->replaceAuditRouteUrl()\n                : route($this->replaceLastRouteSegment(request()->route()->getName(), 'edit'), $this->getKey()),","after":"            'url' => $this->resolveAuditUrl(),"}},{"title":"Double write","description":"create() followed by save()","file":"app/Traits/Auditable.php","line":112,"severity":"medium","suggestion":"Compute message before create","fix":{"start_line":106,"end_line":115,"before":"        $log = AuditLog::create([...]);\n        $log->message = ...;\n        $log->save();","after":"        $log = new AuditLog([...]);\n        $log->message = ...;\n        $log->save();"}}]}

Empty: {"issues":[]}`

func BuildDiffFixPrompt() string {
	base := BuildDiffPrompt()
	base = stripJSONSchemaLines(base)
	return base + fixPromptSuffix
}

func BuildScanFixPrompt() string {
	base := BuildScanPrompt()
	base = stripJSONSchemaLines(base)
	return base + fixPromptSuffix
}

func stripJSONSchemaLines(prompt string) string {
	lines := strings.Split(prompt, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, `{"issues":[{`) || trimmed == `Empty: {"issues":[]}` {
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

// truncate shortens a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const cacheFile = ".larasense-limbo-cache.json"

// CachedIssue represents a single code review finding stored in cache.
type CachedIssue struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Severity    string `json:"severity"`
	Suggestion  string `json:"suggestion"`
}

// Entry represents a cached review result for a single file.
type Entry struct {
	Hash       string        `json:"hash"`
	ReviewedAt time.Time     `json:"reviewed_at"`
	Issues     []CachedIssue `json:"issues,omitempty"`
}

// Cache holds cached file hashes to skip re-reviewing unchanged files.
type Cache struct {
	Files   map[string]Entry `json:"files"`
	dirty   bool
}

// Load reads the cache from disk. Returns empty cache if file doesn't exist.
func Load() *Cache {
	c := &Cache{Files: make(map[string]Entry)}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return c
	}

	json.Unmarshal(data, c)
	if c.Files == nil {
		c.Files = make(map[string]Entry)
	}
	return c
}

// Save writes the cache to disk if it was modified.
func (c *Cache) Save() error {
	if !c.dirty {
		return nil
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache: %w", err)
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// HasChanged checks if a file's diff content has changed since last review.
// Returns true if the file should be reviewed (new or changed).
func (c *Cache) HasChanged(filePath, diffContent string) bool {
	hash := hashContent(diffContent)

	entry, exists := c.Files[filePath]
	if !exists {
		return true
	}

	return entry.Hash != hash
}

// Update records that a file has been reviewed with the given diff content.
func (c *Cache) Update(filePath, diffContent string) {
	c.Files[filePath] = Entry{
		Hash:       hashContent(diffContent),
		ReviewedAt: time.Now(),
	}
	c.dirty = true
}

// UpdateWithIssues records a file review along with the AI-reported issues.
// On subsequent runs, cached issues are returned directly without calling the AI.
func (c *Cache) UpdateWithIssues(filePath, diffContent string, issues []CachedIssue) {
	c.Files[filePath] = Entry{
		Hash:       hashContent(diffContent),
		ReviewedAt: time.Now(),
		Issues:     issues,
	}
	c.dirty = true
}

// GetCachedIssues returns cached issues for a file if the diff is unchanged.
// Returns nil, false if the file is not cached or has changed.
func (c *Cache) GetCachedIssues(filePath, diffContent string) ([]CachedIssue, bool) {
	entry, exists := c.Files[filePath]
	if !exists {
		return nil, false
	}

	hash := hashContent(diffContent)
	if entry.Hash != hash {
		return nil, false
	}

	// Entry exists and hash matches — return cached issues (may be empty slice = no issues found)
	if entry.Issues != nil {
		return entry.Issues, true
	}

	return nil, false
}

// hashContent returns a SHA-256 hash of the content.
func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h)
}

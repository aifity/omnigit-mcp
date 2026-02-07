// Package bodyfilter provides functionality to filter out specific patterns
// from PR/issue body text and git commit messages.
package bodyfilter

import (
	"log"
	"regexp"
	"strings"
)

// Default patterns to filter out from PR/issue bodies and commit messages
var defaultFilterPatterns = []string{
	// Co-Authored-By trailer - commonly added by git but may not be desired in PR description or commit message
	`(?m)^Co-Authored-By:.*$`,
	// Generic PR footer pattern - matches "Pull Request opened by [Tool](url) with guidance from..." format
	// Uses (?s) for single-line mode to match across newlines, and matches the entire line
	`(?s)---\s*Pull Request opened by \[[^\]]+\]\([^)]+\) with guidance from [^\n]+`,
}

// filterPatterns holds the compiled regex patterns
var filterPatterns []*regexp.Regexp

// Initialize with default patterns
func init() {
	SetFilterPatterns(defaultFilterPatterns)
}

// SetFilterPatterns sets the filter patterns from a list of regex strings.
// This allows configuration of custom filter patterns.
// If any pattern fails to compile, it logs an error and skips that pattern.
func SetFilterPatterns(patterns []string) {
	filterPatterns = make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			log.Printf("Warning: failed to compile filter pattern %q: %v", pattern, err)
			continue
		}
		filterPatterns = append(filterPatterns, compiled)
	}
}

// FilterBody removes specific patterns from the body text.
// It checks if the body is not empty, then filters out patterns like:
// - Co-Authored-By: <name> <email>
//
// The function returns the filtered body text with filtered lines removed.
// Empty lines resulting from filtering are preserved to maintain formatting.
func FilterBody(body string) string {
	if body == "" {
		return body
	}

	filtered := body
	for _, pattern := range filterPatterns {
		filtered = pattern.ReplaceAllString(filtered, "")
	}

	// Clean up multiple consecutive blank lines (more than 2 newlines in a row)
	// This helps clean up gaps left by filtered content
	multipleNewlines := regexp.MustCompile(`\n{3,}`)
	filtered = multipleNewlines.ReplaceAllString(filtered, "\n\n")

	// Trim leading and trailing whitespace
	filtered = strings.TrimSpace(filtered)

	return filtered
}


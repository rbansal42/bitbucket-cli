package cmdutil

import "strings"

// TruncateString truncates a string to maxLen characters, replacing newlines
// with spaces and adding "..." if truncated.
func TruncateString(s string, maxLen int) string {
	// Replace newlines with spaces
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	// Collapse multiple spaces
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	s = strings.TrimSpace(s)

	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

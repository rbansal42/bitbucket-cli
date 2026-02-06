package cmdutil

import (
	"fmt"
	"time"
)

// TimeAgo returns a human-readable relative time string for a time.Time value.
// Returns "-" for zero time values.
func TimeAgo(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 30*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case duration < 365*24*time.Hour:
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(duration.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// TimeAgoFromString parses an ISO 8601 / RFC3339 timestamp string and returns
// a human-readable relative time. Returns the raw string on parse failure.
func TimeAgoFromString(isoTime string) string {
	if isoTime == "" {
		return "-"
	}

	t, err := time.Parse(time.RFC3339, isoTime)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000000-07:00", isoTime)
		if err != nil {
			return isoTime
		}
	}

	return TimeAgo(t)
}

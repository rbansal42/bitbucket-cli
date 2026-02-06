package cmdutil

import "github.com/rbansal42/bitbucket-cli/internal/api"

// GetUserDisplayName returns the best available display name for a user.
// Returns "-" if user is nil, falls back through Username → Nickname → "unknown".
func GetUserDisplayName(user *api.User) string {
	if user == nil {
		return "-"
	}
	if user.DisplayName != "" {
		return user.DisplayName
	}
	if user.Username != "" {
		return user.Username
	}
	if user.Nickname != "" {
		return user.Nickname
	}
	return "unknown"
}

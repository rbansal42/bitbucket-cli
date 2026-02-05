package config

import (
	"testing"
)

func TestKeyringKey(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		user     string
		expected string
	}{
		{
			name:     "standard host and user",
			host:     "bitbucket.org",
			user:     "testuser",
			expected: "bitbucket.org:testuser",
		},
		{
			name:     "host with subdomain",
			host:     "api.bitbucket.org",
			user:     "admin",
			expected: "api.bitbucket.org:admin",
		},
		{
			name:     "empty user",
			host:     "bitbucket.org",
			user:     "",
			expected: "bitbucket.org:",
		},
		{
			name:     "empty host",
			host:     "",
			user:     "testuser",
			expected: ":testuser",
		},
		{
			name:     "user with special characters",
			host:     "bitbucket.org",
			user:     "user@example.com",
			expected: "bitbucket.org:user@example.com",
		},
		{
			name:     "user with hyphens and underscores",
			host:     "bitbucket.org",
			user:     "test-user_123",
			expected: "bitbucket.org:test-user_123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := keyringKey(tt.host, tt.user)
			if result != tt.expected {
				t.Errorf("keyringKey(%q, %q) = %q, want %q", tt.host, tt.user, result, tt.expected)
			}
		})
	}
}

func TestLookupEnv(t *testing.T) {
	// Test retrieving an existing environment variable
	t.Setenv("TEST_LOOKUP_VAR", "test_value")

	result := lookupEnv("TEST_LOOKUP_VAR")
	if result != "test_value" {
		t.Errorf("lookupEnv(\"TEST_LOOKUP_VAR\") = %q, want %q", result, "test_value")
	}
}

func TestLookupEnv_NonexistentVar(t *testing.T) {
	// Ensure the variable doesn't exist
	// Using a unique name that won't conflict with actual env vars
	result := lookupEnv("NONEXISTENT_VAR_12345")
	if result != "" {
		t.Errorf("lookupEnv for nonexistent var = %q, want empty string", result)
	}
}

func TestLookupEnv_EmptyValue(t *testing.T) {
	t.Setenv("TEST_EMPTY_VAR", "")

	result := lookupEnv("TEST_EMPTY_VAR")
	if result != "" {
		t.Errorf("lookupEnv for empty var = %q, want empty string", result)
	}
}

func TestGetEnvToken_BBToken(t *testing.T) {
	t.Setenv("BB_TOKEN", "bb-token-value")
	t.Setenv("BITBUCKET_TOKEN", "") // Clear BITBUCKET_TOKEN

	token := getEnvToken()
	if token != "bb-token-value" {
		t.Errorf("getEnvToken() = %q, want %q", token, "bb-token-value")
	}
}

func TestGetEnvToken_BitbucketToken(t *testing.T) {
	t.Setenv("BB_TOKEN", "")
	t.Setenv("BITBUCKET_TOKEN", "bitbucket-token-value")

	token := getEnvToken()
	if token != "bitbucket-token-value" {
		t.Errorf("getEnvToken() = %q, want %q", token, "bitbucket-token-value")
	}
}

func TestGetEnvToken_BBTokenTakesPrecedence(t *testing.T) {
	t.Setenv("BB_TOKEN", "bb-priority-token")
	t.Setenv("BITBUCKET_TOKEN", "bitbucket-fallback-token")

	token := getEnvToken()
	if token != "bb-priority-token" {
		t.Errorf("getEnvToken() = %q, want %q (BB_TOKEN should take precedence)", token, "bb-priority-token")
	}
}

func TestGetEnvToken_NoTokenSet(t *testing.T) {
	t.Setenv("BB_TOKEN", "")
	t.Setenv("BITBUCKET_TOKEN", "")

	token := getEnvToken()
	if token != "" {
		t.Errorf("getEnvToken() with no tokens = %q, want empty string", token)
	}
}

func TestServiceNameConstant(t *testing.T) {
	// Verify the service name constant has expected value
	expected := "bb:bitbucket-cli"
	if ServiceName != expected {
		t.Errorf("ServiceName = %q, want %q", ServiceName, expected)
	}
}

func TestKeyringKey_Format(t *testing.T) {
	// Test that the key format is consistent and uses colon separator
	key := keyringKey("host.example.com", "username")

	// Key should be in format "host:user"
	if key != "host.example.com:username" {
		t.Errorf("keyringKey format incorrect: got %q", key)
	}

	// Verify colon is used as separator
	if key[len("host.example.com")] != ':' {
		t.Error("keyringKey should use ':' as separator between host and user")
	}
}

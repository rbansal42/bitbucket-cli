package config

import (
	"fmt"
	"os"

	"github.com/zalando/go-keyring"
)

const (
	// ServiceName is the name used for keyring storage
	ServiceName = "bb:bitbucket-cli"
)

// KeyringToken represents a token stored in the system keyring
type KeyringToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
}

// keyringKey generates the keyring key for a host and user
func keyringKey(host, user string) string {
	return fmt.Sprintf("%s:%s", host, user)
}

// SetToken stores a token in the system keyring
func SetToken(host, user, token string) error {
	key := keyringKey(host, user)
	return keyring.Set(ServiceName, key, token)
}

// GetToken retrieves a token from the system keyring
func GetToken(host, user string) (string, error) {
	key := keyringKey(host, user)
	token, err := keyring.Get(ServiceName, key)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", fmt.Errorf("no token found for %s@%s", user, host)
		}
		return "", fmt.Errorf("could not retrieve token: %w", err)
	}
	return token, nil
}

// DeleteToken removes a token from the system keyring
func DeleteToken(host, user string) error {
	key := keyringKey(host, user)
	err := keyring.Delete(ServiceName, key)
	if err != nil && err != keyring.ErrNotFound {
		return fmt.Errorf("could not delete token: %w", err)
	}
	return nil
}

// HasToken checks if a token exists in the keyring
func HasToken(host, user string) bool {
	_, err := GetToken(host, user)
	return err == nil
}

// GetTokenFromEnvOrKeyring tries to get a token from environment variable first,
// then falls back to the keyring
func GetTokenFromEnvOrKeyring(host, user string) (string, string, error) {
	// Check environment variable first
	if token := getEnvToken(); token != "" {
		return token, "environment", nil
	}

	// Fall back to keyring
	token, err := GetToken(host, user)
	if err != nil {
		return "", "", err
	}

	return token, "keyring", nil
}

// getEnvToken checks for token in environment variables
func getEnvToken() string {
	// BB_TOKEN takes precedence
	if token := lookupEnv("BB_TOKEN"); token != "" {
		return token
	}

	// Also check BITBUCKET_TOKEN for compatibility
	if token := lookupEnv("BITBUCKET_TOKEN"); token != "" {
		return token
	}

	return ""
}

func lookupEnv(key string) string {
	return os.Getenv(key)
}

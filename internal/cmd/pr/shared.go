package pr

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rbansal42/bitbucket-cli/internal/config"
)

// parsePRNumber parses a PR number from args or returns an error
func parsePRNumber(args []string) (int, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("pull request number is required")
	}

	prNum, err := strconv.Atoi(args[0])
	if err != nil {
		return 0, fmt.Errorf("invalid pull request number: %s", args[0])
	}

	// Validate positive PR number
	if prNum <= 0 {
		return 0, fmt.Errorf("invalid pull request number: must be a positive integer")
	}

	return prNum, nil
}

// openEditor opens the user's preferred editor for text input
func openEditor(initialContent string) (string, error) {
	editor := getEditor()

	// Create temp file
	tmpFile, err := os.CreateTemp("", "bb-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial content
	if initialContent != "" {
		if _, err := tmpFile.WriteString(initialContent); err != nil {
			return "", fmt.Errorf("failed to write to temp file: %w", err)
		}
	}
	tmpFile.Close()

	// Open editor
	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	// Read content back
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read temp file: %w", err)
	}

	return strings.TrimSpace(string(content)), nil
}

// getEditor returns the user's preferred editor
func getEditor() string {
	// Check BB_EDITOR first
	if editor := os.Getenv("BB_EDITOR"); editor != "" {
		return editor
	}

	// Check config
	cfg, err := config.LoadConfig()
	if err == nil && cfg.Editor != "" {
		return cfg.Editor
	}

	// Check standard environment variables
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	// Default to vi
	return "vi"
}

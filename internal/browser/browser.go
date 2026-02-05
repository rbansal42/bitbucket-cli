package browser

import (
	"os"
	"os/exec"
	"runtime"
)

// Open opens the given URL in the default browser
func Open(url string) error {
	// Check for BB_BROWSER environment variable
	if browser := os.Getenv("BB_BROWSER"); browser != "" {
		return exec.Command(browser, url).Start()
	}

	// Check for BROWSER environment variable
	if browser := os.Getenv("BROWSER"); browser != "" {
		return exec.Command(browser, url).Start()
	}

	// Use platform-specific command
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}

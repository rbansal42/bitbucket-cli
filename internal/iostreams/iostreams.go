package iostreams

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// IOStreams provides access to standard input/output streams
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer

	colorEnabled  bool
	is256enabled  bool
	terminalWidth int
}

// New creates a new IOStreams with default stdin/stdout/stderr
func New() *IOStreams {
	io := &IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	// Check if color should be enabled
	io.colorEnabled = io.shouldEnableColor()
	io.is256enabled = io.shouldEnable256Color()

	return io
}

// IsStdoutTTY returns true if stdout is a terminal
func (s *IOStreams) IsStdoutTTY() bool {
	if f, ok := s.Out.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// IsStderrTTY returns true if stderr is a terminal
func (s *IOStreams) IsStderrTTY() bool {
	if f, ok := s.ErrOut.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// IsStdinTTY returns true if stdin is a terminal
func (s *IOStreams) IsStdinTTY() bool {
	if f, ok := s.In.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// ColorEnabled returns true if color output is enabled
func (s *IOStreams) ColorEnabled() bool {
	return s.colorEnabled
}

// Is256ColorEnabled returns true if 256-color output is enabled
func (s *IOStreams) Is256ColorEnabled() bool {
	return s.is256enabled
}

// TerminalWidth returns the width of the terminal, or 80 if not a terminal
func (s *IOStreams) TerminalWidth() int {
	if s.terminalWidth > 0 {
		return s.terminalWidth
	}

	if f, ok := s.Out.(*os.File); ok {
		width, _, err := term.GetSize(int(f.Fd()))
		if err == nil && width > 0 {
			return width
		}
	}

	return 80 // default width
}

func (s *IOStreams) shouldEnableColor() bool {
	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check BB_NO_COLOR environment variable
	if os.Getenv("BB_NO_COLOR") != "" {
		return false
	}

	// Check TERM
	term := os.Getenv("TERM")
	if term == "dumb" {
		return false
	}

	// Enable color if stdout is a TTY
	return s.IsStdoutTTY()
}

func (s *IOStreams) shouldEnable256Color() bool {
	if !s.colorEnabled {
		return false
	}

	term := os.Getenv("TERM")
	return strings.Contains(term, "256color") ||
		strings.Contains(term, "truecolor") ||
		os.Getenv("COLORTERM") != ""
}

// Color codes
const (
	Reset      = "\033[0m"
	Bold       = "\033[1m"
	Red        = "\033[31m"
	Green      = "\033[32m"
	Yellow     = "\033[33m"
	Blue       = "\033[34m"
	Magenta    = "\033[35m"
	Cyan       = "\033[36m"
	White      = "\033[37m"
	BoldRed    = "\033[1;31m"
	BoldGreen  = "\033[1;32m"
	BoldYellow = "\033[1;33m"
	BoldBlue   = "\033[1;34m"
)

// ColorFunc returns a function that wraps text in color codes if color is enabled
func (s *IOStreams) ColorFunc(color string) func(string) string {
	if !s.colorEnabled {
		return func(text string) string { return text }
	}
	return func(text string) string {
		return color + text + Reset
	}
}

// Success prints a success message (green checkmark)
func (s *IOStreams) Success(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	if s.colorEnabled {
		fmt.Fprintf(s.Out, "%s%s %s%s\n", Green, "✓", msg, Reset)
	} else {
		fmt.Fprintf(s.Out, "✓ %s\n", msg)
	}
}

// Error prints an error message (red X)
func (s *IOStreams) Error(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	if s.colorEnabled {
		fmt.Fprintf(s.ErrOut, "%s%s %s%s\n", Red, "✗", msg, Reset)
	} else {
		fmt.Fprintf(s.ErrOut, "✗ %s\n", msg)
	}
}

// Warning prints a warning message (yellow !)
func (s *IOStreams) Warning(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	if s.colorEnabled {
		fmt.Fprintf(s.ErrOut, "%s%s %s%s\n", Yellow, "!", msg, Reset)
	} else {
		fmt.Fprintf(s.ErrOut, "! %s\n", msg)
	}
}

// Info prints an info message
func (s *IOStreams) Info(format string, a ...interface{}) {
	fmt.Fprintf(s.Out, format+"\n", a...)
}

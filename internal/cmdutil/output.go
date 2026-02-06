package cmdutil

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// PrintJSON marshals v as indented JSON and writes it to streams.Out.
func PrintJSON(streams *iostreams.IOStreams, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Fprintln(streams.Out, string(data))
	return nil
}

// PrintTableHeader writes a bold header line to a tabwriter if color is enabled,
// otherwise writes a plain header.
func PrintTableHeader(streams *iostreams.IOStreams, w *tabwriter.Writer, header string) {
	if streams.ColorEnabled() {
		fmt.Fprintln(w, iostreams.Bold+header+iostreams.Reset)
	} else {
		fmt.Fprintln(w, header)
	}
}

// ConfirmPrompt reads a line from reader and returns true if user typed y/yes.
func ConfirmPrompt(reader io.Reader) bool {
	scanner := bufio.NewScanner(reader)
	if scanner.Scan() {
		input := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return input == "y" || input == "yes"
	}
	return false
}

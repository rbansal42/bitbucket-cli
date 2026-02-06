package completion

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// NewCmdCompletion creates the completion command and its subcommands
func NewCmdCompletion(streams *iostreams.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion <shell>",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for bb.

The completion script must be evaluated to provide interactive completion.
This can be done by sourcing it in your shell configuration.

For examples of loading completions, run:
  bb completion bash --help
  bb completion zsh --help
  bb completion fish --help
  bb completion powershell --help`,
		Example: `  # Generate bash completion
  bb completion bash

  # Generate zsh completion
  bb completion zsh

  # Generate fish completion
  bb completion fish

  # Generate PowerShell completion
  bb completion powershell`,
	}

	cmd.AddCommand(NewCmdBash(streams))
	cmd.AddCommand(NewCmdZsh(streams))
	cmd.AddCommand(NewCmdFish(streams))
	cmd.AddCommand(NewCmdPowerShell(streams))

	return cmd
}

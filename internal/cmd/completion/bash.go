package completion

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// NewCmdBash creates the bash completion command
func NewCmdBash(streams *iostreams.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "bash",
		Short: "Generate bash completion script",
		Long: `Generate the autocompletion script for bash.

To load completions in your current shell session:

    source <(bb completion bash)

To load completions for every new session, execute once:

Linux:
    bb completion bash > /etc/bash_completion.d/bb

macOS:
    bb completion bash > $(brew --prefix)/etc/bash_completion.d/bb

You will need to start a new shell for this setup to take effect.`,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenBashCompletionV2(streams.Out, true)
		},
	}
}

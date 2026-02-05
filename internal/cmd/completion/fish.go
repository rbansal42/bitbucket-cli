package completion

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// NewCmdFish creates the fish completion command
func NewCmdFish(streams *iostreams.IOStreams) *cobra.Command {
	var noDescriptions bool

	cmd := &cobra.Command{
		Use:   "fish",
		Short: "Generate fish completion script",
		Long: `Generate the autocompletion script for fish.

To load completions in your current shell session:

    bb completion fish | source

To load completions for every new session, execute once:

    bb completion fish > ~/.config/fish/completions/bb.fish

You will need to start a new shell for this setup to take effect.`,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenFishCompletion(streams.Out, !noDescriptions)
		},
	}

	cmd.Flags().BoolVar(&noDescriptions, "no-descriptions", false, "Disable completion descriptions")

	return cmd
}

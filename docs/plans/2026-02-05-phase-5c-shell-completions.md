# Phase 5c: Shell Completions Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add shell completion support for bash, zsh, fish, and PowerShell to the bb CLI.

**Architecture:** Use Cobra's built-in completion generation with a dedicated `completion` command. Provide subcommands for each shell type.

**Tech Stack:** Go, Cobra CLI (spf13/cobra), shell completion scripts

---

## Overview

Cobra provides built-in shell completion generation. We'll create a `completion` command that generates completion scripts for:
- bash
- zsh
- fish
- powershell

Users can then source these scripts in their shell config.

---

## Task 1: Create Completion Command Package

**Files:**
- Create: `internal/cmd/completion/completion.go`

### completion.go

```go
package completion

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
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
```

---

## Task 2: Create Bash Completion Subcommand

**Files:**
- Create: `internal/cmd/completion/bash.go`

### bash.go

```go
package completion

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
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
```

---

## Task 3: Create Zsh Completion Subcommand

**Files:**
- Create: `internal/cmd/completion/zsh.go`

### zsh.go

```go
package completion

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

// NewCmdZsh creates the zsh completion command
func NewCmdZsh(streams *iostreams.IOStreams) *cobra.Command {
	var noDescriptions bool

	cmd := &cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completion script",
		Long: `Generate the autocompletion script for zsh.

To load completions in your current shell session:

    source <(bb completion zsh)

To load completions for every new session, execute once:

Linux:
    bb completion zsh > "${fpath[1]}/_bb"

macOS:
    bb completion zsh > $(brew --prefix)/share/zsh/site-functions/_bb

You will need to start a new shell for this setup to take effect.

If shell completion is not already enabled in your environment, you will need
to enable it. You can execute the following once:

    echo "autoload -U compinit; compinit" >> ~/.zshrc`,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if noDescriptions {
				return cmd.Root().GenZshCompletionNoDesc(streams.Out)
			}
			return cmd.Root().GenZshCompletion(streams.Out)
		},
	}

	cmd.Flags().BoolVar(&noDescriptions, "no-descriptions", false, "Disable completion descriptions")

	return cmd
}
```

---

## Task 4: Create Fish Completion Subcommand

**Files:**
- Create: `internal/cmd/completion/fish.go`

### fish.go

```go
package completion

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
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
```

---

## Task 5: Create PowerShell Completion Subcommand

**Files:**
- Create: `internal/cmd/completion/powershell.go`

### powershell.go

```go
package completion

import (
	"github.com/spf13/cobra"

	"github.com/rbansal42/bb/internal/iostreams"
)

// NewCmdPowerShell creates the powershell completion command
func NewCmdPowerShell(streams *iostreams.IOStreams) *cobra.Command {
	var noDescriptions bool

	cmd := &cobra.Command{
		Use:   "powershell",
		Short: "Generate PowerShell completion script",
		Long: `Generate the autocompletion script for PowerShell.

To load completions in your current shell session:

    bb completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your PowerShell profile.`,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if noDescriptions {
				return cmd.Root().GenPowerShellCompletion(streams.Out)
			}
			return cmd.Root().GenPowerShellCompletionWithDesc(streams.Out)
		},
	}

	cmd.Flags().BoolVar(&noDescriptions, "no-descriptions", false, "Disable completion descriptions")

	return cmd
}
```

---

## Task 6: Register Completion Command in Root

**Files:**
- Modify: `internal/cmd/root.go`

### Changes

Add import:
```go
"github.com/rbansal42/bb/internal/cmd/completion"
```

Add command registration in `init()`:
```go
rootCmd.AddCommand(completion.NewCmdCompletion(GetStreams()))
```

---

## Task 7: Build and Test Completions

### Build
```bash
go build ./...
```

### Test Bash Completion
```bash
./bb completion bash > /tmp/bb_completion.bash
source /tmp/bb_completion.bash
bb <TAB>  # Should show commands
bb pr <TAB>  # Should show pr subcommands
```

### Test Zsh Completion
```bash
./bb completion zsh > /tmp/_bb
source /tmp/_bb
bb <TAB>  # Should show commands with descriptions
```

### Test Fish Completion
```bash
./bb completion fish > /tmp/bb.fish
source /tmp/bb.fish
bb <TAB>  # Should show commands
```

---

## Summary

| Task | Files | Description |
|------|-------|-------------|
| 1 | completion.go | Parent completion command |
| 2 | bash.go | Bash completion subcommand |
| 3 | zsh.go | Zsh completion subcommand |
| 4 | fish.go | Fish completion subcommand |
| 5 | powershell.go | PowerShell completion subcommand |
| 6 | root.go | Register completion command |
| 7 | - | Build and test |

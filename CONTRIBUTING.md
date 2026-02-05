# Contributing to bb CLI

Thank you for your interest in contributing to `bb`, the Bitbucket Cloud CLI! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md) to help maintain a welcoming and inclusive community.

## Development Setup

### Prerequisites

- **Go 1.21+** - [Download and install Go](https://go.dev/dl/)
- **Git** - For version control

### Getting Started

1. **Fork the repository** on GitHub

2. **Clone your fork:**
   ```bash
   git clone https://github.com/YOUR_USERNAME/bitbucket-cli.git
   cd bitbucket-cli
   ```

3. **Build the CLI:**
   ```bash
   go build ./cmd/bb
   ```

4. **Run tests:**
   ```bash
   go test ./...
   ```

5. **Run the CLI locally:**
   ```bash
   ./bb --help
   ```

## Project Structure

```
bitbucket-cli/
├── cmd/bb/           # Main entry point
│   └── main.go       # Application bootstrap
├── internal/
│   ├── api/          # Bitbucket API client
│   ├── cmd/          # Command implementations
│   ├── cmdutil/      # Shared command utilities
│   └── config/       # Configuration handling
├── go.mod
└── go.sum
```

| Directory | Purpose |
|-----------|---------|
| `cmd/bb/` | Main entry point and CLI initialization |
| `internal/api/` | Bitbucket Cloud API client and types |
| `internal/cmd/` | Individual command implementations (pr, repo, etc.) |
| `internal/config/` | Configuration loading, storage, and authentication |
| `internal/cmdutil/` | Shared utilities for commands (formatting, prompts, etc.) |

## Adding New Commands

We use [Cobra](https://github.com/spf13/cobra) for command management. To add a new command:

1. **Create a new file** in `internal/cmd/` (or appropriate subdirectory):

   ```go
   // internal/cmd/mycommand/mycommand.go
   package mycommand

   import (
       "github.com/spf13/cobra"
   )

   func NewCmdMyCommand() *cobra.Command {
       cmd := &cobra.Command{
           Use:   "mycommand",
           Short: "Brief description of the command",
           Long:  `Longer description explaining the command in detail.`,
           RunE: func(cmd *cobra.Command, args []string) error {
               // Command implementation
               return nil
           },
       }

       // Add flags
       cmd.Flags().StringP("flag-name", "f", "", "Flag description")

       return cmd
   }
   ```

2. **Register the command** in the parent command or root command.

3. **Add tests** for your command in a `_test.go` file.

## Code Style Guidelines

### Formatting

- Run `gofmt` on all code before committing:
  ```bash
  gofmt -w .
  ```

- Run `go vet` to catch common issues:
  ```bash
  go vet ./...
  ```

### Best Practices

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Keep functions focused and small
- Use meaningful variable and function names
- Add comments for exported functions and types
- Handle errors explicitly; avoid ignoring them

### Linting (Optional)

We recommend using `golangci-lint` for comprehensive linting:
```bash
golangci-lint run
```

## Testing Requirements

- **All new features must include tests**
- **Bug fixes should include regression tests**
- Tests should be in `_test.go` files alongside the code they test

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./internal/cmd/pr/...
```

### Writing Tests

```go
func TestMyFunction(t *testing.T) {
    // Arrange
    input := "test"

    // Act
    result := MyFunction(input)

    // Assert
    if result != expected {
        t.Errorf("MyFunction(%q) = %q; want %q", input, result, expected)
    }
}
```

## Pull Request Process

### Before You Start

1. Check existing issues and PRs to avoid duplicate work
2. For significant changes, open an issue first to discuss the approach

### Creating a Pull Request

1. **Create a feature branch:**
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/issue-description
   ```

2. **Make your changes** with clear, focused commits

3. **Commit message format:**
   ```
   type: short description

   Longer description if needed. Explain what and why,
   not how (the code shows how).

   Fixes #123
   ```

   Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`

   Examples:
   ```
   feat: add support for PR templates

   fix: handle empty repository list gracefully

   docs: update installation instructions
   ```

4. **Push your branch:**
   ```bash
   git push origin feature/your-feature-name
   ```

5. **Open a Pull Request** with the following information:

   ```markdown
   ## Description
   Brief description of the changes.

   ## Type of Change
   - [ ] Bug fix
   - [ ] New feature
   - [ ] Breaking change
   - [ ] Documentation update

   ## Testing
   Describe how you tested your changes.

   ## Checklist
   - [ ] Code follows the project's style guidelines
   - [ ] Tests added/updated for changes
   - [ ] Documentation updated if needed
   - [ ] `go test ./...` passes
   - [ ] `go vet ./...` reports no issues
   ```

### Review Process

- PRs require at least one maintainer approval
- Address review feedback promptly
- Keep PRs focused; split large changes into smaller PRs

## Issue Reporting Guidelines

### Bug Reports

When reporting a bug, include:

- **bb version:** Output of `bb --version`
- **Go version:** Output of `go version`
- **Operating system:** e.g., macOS 14.0, Ubuntu 22.04
- **Steps to reproduce:** Clear, numbered steps
- **Expected behavior:** What you expected to happen
- **Actual behavior:** What actually happened
- **Error messages:** Full error output if applicable

### Feature Requests

When requesting a feature, include:

- **Use case:** Why do you need this feature?
- **Proposed solution:** How do you envision it working?
- **Alternatives considered:** Other approaches you've thought about

## Release Process (Maintainers)

1. **Update version** in relevant files

2. **Update CHANGELOG.md** with release notes

3. **Create a release commit:**
   ```bash
   git commit -am "chore: release vX.Y.Z"
   ```

4. **Tag the release:**
   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   git push origin main --tags
   ```

5. **Create GitHub release** with changelog notes

6. **Verify** release artifacts are built and published correctly

---

## Questions?

If you have questions about contributing, feel free to:

- Open a [Discussion](https://github.com/YOUR_ORG/bitbucket-cli/discussions)
- Ask in an existing related issue

Thank you for contributing!

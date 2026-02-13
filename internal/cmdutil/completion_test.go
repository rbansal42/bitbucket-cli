package cmdutil

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestStaticFlagCompletion(t *testing.T) {
	values := []string{"open", "merged", "declined"}
	fn := StaticFlagCompletion(values)

	cmd := &cobra.Command{}
	result, directive := fn(cmd, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}

	if len(result) != len(values) {
		t.Errorf("expected %d values, got %d", len(values), len(result))
	}

	for i, v := range values {
		if result[i] != v {
			t.Errorf("expected result[%d] = %q, got %q", i, v, result[i])
		}
	}
}

func TestStaticFlagCompletionFiltering(t *testing.T) {
	values := []string{"open", "merged", "declined"}
	fn := StaticFlagCompletion(values)

	cmd := &cobra.Command{}

	tests := []struct {
		name       string
		toComplete string
		expected   []string
	}{
		{
			name:       "prefix match lowercase",
			toComplete: "o",
			expected:   []string{"open"},
		},
		{
			name:       "prefix match uppercase (case-insensitive)",
			toComplete: "O",
			expected:   []string{"open"},
		},
		{
			name:       "prefix match multiple",
			toComplete: "m",
			expected:   []string{"merged"},
		},
		{
			name:       "prefix match d",
			toComplete: "d",
			expected:   []string{"declined"},
		},
		{
			name:       "no match",
			toComplete: "x",
			expected:   nil,
		},
		{
			name:       "exact match",
			toComplete: "open",
			expected:   []string{"open"},
		},
		{
			name:       "case-insensitive full match",
			toComplete: "OPEN",
			expected:   []string{"open"},
		},
		{
			name:       "partial match me",
			toComplete: "me",
			expected:   []string{"merged"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, directive := fn(cmd, nil, tt.toComplete)

			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d results, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, v := range tt.expected {
				if result[i] != v {
					t.Errorf("expected result[%d] = %q, got %q", i, v, result[i])
				}
			}
		})
	}
}

func TestStaticFlagCompletionEmpty(t *testing.T) {
	values := []string{"alpha", "beta", "gamma"}
	fn := StaticFlagCompletion(values)

	cmd := &cobra.Command{}
	result, directive := fn(cmd, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}

	if len(result) != len(values) {
		t.Errorf("expected %d values when toComplete is empty, got %d", len(values), len(result))
	}

	for i, v := range values {
		if result[i] != v {
			t.Errorf("expected result[%d] = %q, got %q", i, v, result[i])
		}
	}
}

func TestCompletionCtx(t *testing.T) {
	ctx, cancel := completionCtx()
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context to have a deadline")
	}

	if deadline.IsZero() {
		t.Error("expected non-zero deadline")
	}
}

package pr

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/rbansal42/bitbucket-cli/internal/api"
	"github.com/rbansal42/bitbucket-cli/internal/iostreams"
)

// mkComment builds a PRComment for tests. A non-zero parentID marks it as a reply.
func mkComment(id, parentID int64, author, body string, created time.Time) api.PRComment {
	var c api.PRComment
	c.ID = id
	c.Content.Raw = body
	c.User.DisplayName = author
	c.CreatedOn = created
	if parentID != 0 {
		c.Parent = &struct {
			ID int64 `json:"id"`
		}{ID: parentID}
	}
	return c
}

func TestDisplayCommentsThreading(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// Deliberately out of order and with the reply before its parent in the slice.
	comments := []api.PRComment{
		mkComment(3, 1, "Carol", "reply to A", t0.Add(2*time.Hour)),
		mkComment(2, 0, "Bob", "top level B", t0.Add(time.Hour)),
		mkComment(1, 0, "Alice", "top level A", t0),
	}

	streams := iostreams.New()
	var buf bytes.Buffer
	streams.Out = &buf

	if err := displayComments(streams, comments, 42); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()

	// Header shows the total comment count.
	if !strings.Contains(out, "pull request #42 (3 total)") {
		t.Errorf("expected header with total count, got:\n%s", out)
	}

	// The reply body is nested one level deeper (4-space indent) than a
	// top-level body (2-space indent). This is color-independent.
	if !strings.Contains(out, "\n    reply to A") {
		t.Errorf("expected reply body indented under its parent, got:\n%s", out)
	}
	if !strings.Contains(out, "\n  top level A") {
		t.Errorf("expected top-level body at base indent, got:\n%s", out)
	}

	// Ordering: top-level comments chronological (A before B), and a reply
	// rendered immediately under its parent (A) rather than under B.
	idxA := strings.Index(out, "top level A")
	idxReply := strings.Index(out, "reply to A")
	idxB := strings.Index(out, "top level B")
	if !(idxA >= 0 && idxReply > idxA && idxB > idxReply) {
		t.Errorf("expected order A, reply-to-A, B; got A=%d reply=%d B=%d\n%s", idxA, idxReply, idxB, out)
	}
}

func TestDisplayCommentsEmpty(t *testing.T) {
	streams := iostreams.New()
	var buf bytes.Buffer
	streams.Out = &buf

	if err := displayComments(streams, nil, 7); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No comments on pull request #7") {
		t.Errorf("expected empty-state message, got %q", buf.String())
	}
}

func TestResolveCommentPRNumber(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		want     int
		wantErr  bool
	}{
		{name: "valid number", selector: "123", want: 123},
		{name: "zero is invalid", selector: "0", wantErr: true},
		{name: "negative is invalid", selector: "-5", wantErr: true},
		{name: "non-numeric is invalid", selector: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveCommentPRNumber(nil, "workspace", "repo", tt.selector)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for selector %q, got nil", tt.selector)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveCommentPRNumber(%q) = %d; want %d", tt.selector, got, tt.want)
			}
		})
	}
}

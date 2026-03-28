package notify

import (
	"strings"
	"testing"

	"github.com/hl/brr/internal/engine"
)

func TestFormatComplete(t *testing.T) {
	title, body := format(&engine.Result{Reason: engine.ReasonComplete})
	if title != "brr — complete" {
		t.Errorf("unexpected title: %q", title)
	}
	if body != "All tasks complete." {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestFormatApprovalWithContent(t *testing.T) {
	title, body := format(&engine.Result{
		Reason:          engine.ReasonApproval,
		ApprovalContent: "Please review the migration plan",
	})
	if title != "brr — approval needed" {
		t.Errorf("unexpected title: %q", title)
	}
	if body != "Please review the migration plan" {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestFormatApprovalWithoutContent(t *testing.T) {
	title, body := format(&engine.Result{Reason: engine.ReasonApproval})
	if title != "brr — approval needed" {
		t.Errorf("unexpected title: %q", title)
	}
	if body != "A task needs human approval." {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestFormatApprovalTruncatesLongContent(t *testing.T) {
	long := strings.Repeat("word ", 100) // 500 chars
	_, body := format(&engine.Result{
		Reason:          engine.ReasonApproval,
		ApprovalContent: long,
	})
	if len(body) > 260 {
		t.Errorf("expected body to be truncated, got %d bytes", len(body))
	}
	if !strings.HasSuffix(body, "…") {
		t.Error("expected truncation marker")
	}
}

func TestFormatMaxIterations(t *testing.T) {
	title, body := format(&engine.Result{Reason: engine.ReasonMaxIterations})
	if title != "brr — max iterations" {
		t.Errorf("unexpected title: %q", title)
	}
	if body != "Maximum iteration count reached." {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestFormatFailStreak(t *testing.T) {
	title, body := format(&engine.Result{Reason: engine.ReasonFailStreak})
	if title != "brr — stopped" {
		t.Errorf("unexpected title: %q", title)
	}
	if body != "Too many consecutive failures." {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestTruncateShort(t *testing.T) {
	s := "hello"
	if truncate(s, 256) != "hello" {
		t.Errorf("short string should not be truncated")
	}
}

func TestTruncateLong(t *testing.T) {
	s := "the quick brown fox jumps over the lazy dog"
	result := truncate(s, 15)
	if !strings.HasSuffix(result, "…") {
		t.Error("expected truncation marker")
	}
	// Should break at word boundary — "the quick brown" is 15 chars,
	// last space before index 15 is at 9, so "the quick…"
	prefix := strings.TrimSuffix(result, "…")
	if strings.ContainsRune(prefix, ' ') {
		// Ensure it doesn't end mid-word
		lastSpace := strings.LastIndex(s[:15], " ")
		if len(prefix) != lastSpace {
			t.Errorf("expected word-boundary truncation, got %q", result)
		}
	}
}

package gitops

import (
	"errors"
	"strings"
	"testing"
)

func TestSanitizeText(t *testing.T) {
	input := "clone https://x-access-token:abc123@github.com/runlix/repo.git"
	got := SanitizeText(input)

	if strings.Contains(got, "abc123") {
		t.Fatalf("expected token to be redacted, got: %s", got)
	}
	if !strings.Contains(got, "https://x-access-token:***@github.com/runlix/repo.git") {
		t.Fatalf("unexpected redacted output: %s", got)
	}
}

func TestSanitizeArgs(t *testing.T) {
	args := []string{"clone", "https://x-access-token:secret@github.com/org/repo.git"}
	got := sanitizeArgs(args)

	if strings.Contains(strings.Join(got, " "), "secret") {
		t.Fatalf("expected sanitized args, got: %v", got)
	}
}

func TestFormatGitErrorRedactsArgsAndStderr(t *testing.T) {
	err := formatGitError(
		[]string{"clone", "https://x-access-token:s3cr3t@github.com/org/repo.git"},
		errors.New("exit status 128"),
		"fatal: could not read Username for 'https://x-access-token:s3cr3t@github.com': terminal prompts disabled",
	)

	msg := err.Error()
	if strings.Contains(msg, "s3cr3t") {
		t.Fatalf("expected redaction in formatted error, got: %s", msg)
	}
	if !strings.Contains(msg, "***") {
		t.Fatalf("expected redaction marker in formatted error, got: %s", msg)
	}
}

func TestSanitizeTextNoChange(t *testing.T) {
	input := "git status --porcelain"
	if got := SanitizeText(input); got != input {
		t.Fatalf("expected unchanged text, got: %s", got)
	}
}

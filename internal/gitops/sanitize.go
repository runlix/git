package gitops

import (
	"fmt"
	"regexp"
	"strings"
)

var credentialURLPattern = regexp.MustCompile(`https://([^\s/@:]+):([^@\s/]+)@`)

// SanitizeText removes credential secrets from URL-like content.
func SanitizeText(s string) string {
	if s == "" {
		return s
	}
	return credentialURLPattern.ReplaceAllString(s, "https://$1:***@")
}

func sanitizeArgs(args []string) []string {
	out := make([]string, len(args))
	for i, arg := range args {
		out[i] = SanitizeText(arg)
	}
	return out
}

func formatGitError(args []string, runErr error, stderr string) error {
	safeArgs := strings.Join(sanitizeArgs(args), " ")
	safeStderr := strings.TrimSpace(SanitizeText(stderr))
	return fmt.Errorf("git %s failed: %w stderr=%s", safeArgs, runErr, safeStderr)
}

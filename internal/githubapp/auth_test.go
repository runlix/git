package githubapp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectTokenInHTTPSRepoURL(t *testing.T) {
	got, err := InjectTokenInHTTPSRepoURL("https://github.com/runlix/git.git", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "x-access-token:abc123") {
		t.Fatalf("expected tokenized url, got: %s", got)
	}
}

func TestInjectTokenInHTTPSRepoURLRejectsNonHTTPS(t *testing.T) {
	if _, err := InjectTokenInHTTPSRepoURL("ssh://github.com/runlix/git.git", "abc"); err == nil {
		t.Fatal("expected error for non-https scheme")
	}
}

func TestGetInstallationToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/app/installations/42/access_tokens" {
			http.NotFound(w, r)
			return
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Fatalf("missing bearer auth header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"token-123"}`))
	}))
	defer server.Close()

	t.Setenv("GITHUB_API_URL", server.URL)
	keyFile := writeTestPrivateKey(t)

	token, err := GetInstallationToken("1", "42", keyFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "token-123" {
		t.Fatalf("unexpected token: %s", token)
	}
}

func TestGetInstallationTokenHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	t.Setenv("GITHUB_API_URL", server.URL)
	keyFile := writeTestPrivateKey(t)

	_, err := GetInstallationToken("1", "42", keyFile)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status=401") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseInstallationTokenResponse(t *testing.T) {
	token, err := parseInstallationTokenResponse(strings.NewReader(`{"token":"x"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "x" {
		t.Fatalf("unexpected token: %s", token)
	}
}

func writeTestPrivateKey(t *testing.T) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
	pemBytes := pem.EncodeToMemory(block)

	dir := t.TempDir()
	path := filepath.Join(dir, "app.pem")
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	return path
}

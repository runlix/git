package sync

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseCSV(t *testing.T) {
	got := ParseCSV(" configuration.yaml, ../bad ,packages,, . ,themes ")
	want := []string{"configuration.yaml", "packages", "themes"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected parse result: got=%v want=%v", got, want)
	}
}

func TestCopyAllowlistedFromRepoToConfig(t *testing.T) {
	repoDir := t.TempDir()
	configDir := t.TempDir()

	mustWriteFile(t, filepath.Join(repoDir, "configuration.yaml"), "name: test\n")
	mustWriteFile(t, filepath.Join(repoDir, "packages", "a.yaml"), "a: 1\n")
	mustWriteFile(t, filepath.Join(repoDir, "ignored.txt"), "ignore\n")

	changed, err := CopyAllowlistedFromRepoToConfig(
		repoDir,
		configDir,
		[]string{"configuration.yaml"},
		[]string{"packages"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed < 2 {
		t.Fatalf("expected changed >= 2, got %d", changed)
	}

	mustFileExists(t, filepath.Join(configDir, "configuration.yaml"))
	mustFileExists(t, filepath.Join(configDir, "packages", "a.yaml"))
	mustNotExist(t, filepath.Join(configDir, "ignored.txt"))
}

func TestCopyAllowlistedFromConfigToRepo(t *testing.T) {
	repoDir := t.TempDir()
	configDir := t.TempDir()

	mustWriteFile(t, filepath.Join(configDir, "scripts.yaml"), "script: {}\n")
	mustWriteFile(t, filepath.Join(configDir, "themes", "theme.yaml"), "theme: dark\n")

	changed, err := CopyAllowlistedFromConfigToRepo(
		configDir,
		repoDir,
		[]string{"scripts.yaml"},
		[]string{"themes"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed < 2 {
		t.Fatalf("expected changed >= 2, got %d", changed)
	}

	mustFileExists(t, filepath.Join(repoDir, "scripts.yaml"))
	mustFileExists(t, filepath.Join(repoDir, "themes", "theme.yaml"))
}

func TestCopyDirSkipsDotGit(t *testing.T) {
	repoDir := t.TempDir()
	configDir := t.TempDir()

	mustWriteFile(t, filepath.Join(repoDir, "packages", ".git", "HEAD"), "ref: main\n")
	mustWriteFile(t, filepath.Join(repoDir, "packages", "real.yaml"), "v: 1\n")

	_, err := CopyAllowlistedFromRepoToConfig(repoDir, configDir, nil, []string{"packages"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mustFileExists(t, filepath.Join(configDir, "packages", "real.yaml"))
	mustNotExist(t, filepath.Join(configDir, "packages", ".git", "HEAD"))
}

func TestRemoveDeniedPaths(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "secrets.yaml"), "secret\n")
	mustWriteFile(t, filepath.Join(root, ".storage", "state"), "{}\n")

	removed, err := RemoveDeniedPaths(root, []string{"secrets.yaml", ".storage", "missing"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed != 2 {
		t.Fatalf("expected removed=2 got=%d", removed)
	}
	mustNotExist(t, filepath.Join(root, "secrets.yaml"))
	mustNotExist(t, filepath.Join(root, ".storage"))
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func mustFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %s err=%v", path, err)
	}
}

func mustNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected path not to exist: %s", path)
	}
}

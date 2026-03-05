package sync

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
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

func TestCopyAllowlistedFromConfigToRepo_SymlinkToDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink setup varies on windows")
	}
	configDir := t.TempDir()
	repoDir := t.TempDir()

	targetDir := filepath.Join(configDir, "custom_components", "spook", "integrations", "spook_inverse")
	mustWriteFile(t, filepath.Join(targetDir, "manifest.json"), "{}\n")
	mustSymlink(t, targetDir, filepath.Join(configDir, "custom_components", "spook_inverse"))

	changed, err := CopyAllowlistedFromConfigToRepo(
		configDir,
		repoDir,
		nil,
		[]string{"custom_components"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed < 1 {
		t.Fatalf("expected changed >= 1, got %d", changed)
	}
	mustFileExists(t, filepath.Join(repoDir, "custom_components", "spook_inverse", "manifest.json"))
}

func TestCopyAllowlistedFromConfigToRepo_BrokenSymlinkSkipped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink setup varies on windows")
	}
	configDir := t.TempDir()
	repoDir := t.TempDir()

	mustSymlink(t, filepath.Join(configDir, "missing"), filepath.Join(configDir, "custom_components"))

	_, err := CopyAllowlistedFromConfigToRepo(
		configDir,
		repoDir,
		nil,
		[]string{"custom_components"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mustNotExist(t, filepath.Join(repoDir, "custom_components"))
}

func TestCopyAllowlistedFromConfigToRepo_OutOfRootSymlinkSkipped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink setup varies on windows")
	}
	base := t.TempDir()
	configDir := filepath.Join(base, "config")
	repoDir := filepath.Join(base, "repo")
	outside := filepath.Join(base, "outside")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	mustWriteFile(t, filepath.Join(outside, "secret.txt"), "sensitive\n")
	mustSymlink(t, filepath.Join(outside, "secret.txt"), filepath.Join(configDir, "custom_components"))

	_, err := CopyAllowlistedFromConfigToRepo(
		configDir,
		repoDir,
		nil,
		[]string{"custom_components"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mustNotExist(t, filepath.Join(repoDir, "custom_components"))
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

func mustSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatalf("mkdir link parent: %v", err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
}

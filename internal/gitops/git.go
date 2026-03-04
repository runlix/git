package gitops

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func PrepareRepo(repoDir, repoURL, repoRef string) error {
	if err := os.MkdirAll(filepath.Dir(repoDir), 0o755); err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err != nil {
		if os.IsNotExist(err) {
			if err := run("", "clone", repoURL, repoDir); err != nil {
				return fmt.Errorf("clone: %w", err)
			}
		} else {
			return err
		}
	}

	if err := run(repoDir, "remote", "set-url", "origin", repoURL); err != nil {
		return fmt.Errorf("set origin url: %w", err)
	}
	if err := run(repoDir, "fetch", "--prune", "origin"); err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	if err := run(repoDir, "checkout", "-B", repoRef, "origin/"+repoRef); err != nil {
		if err2 := run(repoDir, "checkout", "-B", repoRef); err2 != nil {
			return fmt.Errorf("checkout: %w", err)
		}
	}
	if err := run(repoDir, "reset", "--hard", "origin/"+repoRef); err != nil {
		return fmt.Errorf("reset: %w", err)
	}

	return nil
}

func ConfigureAuthor(repoDir, name, email string) error {
	if err := run(repoDir, "config", "user.name", name); err != nil {
		return err
	}
	if err := run(repoDir, "config", "user.email", email); err != nil {
		return err
	}
	return nil
}

func AddAll(repoDir string) error {
	return run(repoDir, "add", "-A")
}

func Commit(repoDir, message string) error {
	return run(repoDir, "commit", "-m", message)
}

func Push(repoDir, remote, refspec string) error {
	return run(repoDir, "push", remote, refspec)
}

func RevParseHead(repoDir string) (string, error) {
	out, err := runOutput(repoDir, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func StatusPorcelain(repoDir string) (string, error) {
	return runOutput(repoDir, "status", "--porcelain")
}

func runOutput(dir string, args ...string) (string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(), "HOME=/tmp")

	if err := cmd.Run(); err != nil {
		return "", formatGitError(args, err, stderr.String())
	}

	return stdout.String(), nil
}

func run(dir string, args ...string) error {
	_, err := runOutput(dir, args...)
	return err
}

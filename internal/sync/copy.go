package sync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ParseCSV(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		clean := filepath.Clean(t)
		if clean == "." || strings.HasPrefix(clean, "..") {
			continue
		}
		out = append(out, clean)
	}
	return out
}

func CopyAllowlistedFromRepoToConfig(repoDir, configDir string, files, dirs []string) (int, error) {
	changed := 0

	for _, rel := range files {
		src := filepath.Join(repoDir, rel)
		dst := filepath.Join(configDir, rel)
		n, err := copyFileIfExists(src, dst)
		if err != nil {
			return changed, err
		}
		changed += n
	}

	for _, rel := range dirs {
		src := filepath.Join(repoDir, rel)
		dst := filepath.Join(configDir, rel)
		n, err := copyDirIfExists(src, dst)
		if err != nil {
			return changed, err
		}
		changed += n
	}

	return changed, nil
}

func CopyAllowlistedFromConfigToRepo(configDir, repoDir string, files, dirs []string) (int, error) {
	changed := 0

	for _, rel := range files {
		src := filepath.Join(configDir, rel)
		dst := filepath.Join(repoDir, rel)
		n, err := copyFileIfExists(src, dst)
		if err != nil {
			return changed, err
		}
		changed += n
	}

	for _, rel := range dirs {
		src := filepath.Join(configDir, rel)
		dst := filepath.Join(repoDir, rel)
		n, err := copyDirIfExists(src, dst)
		if err != nil {
			return changed, err
		}
		changed += n
	}

	return changed, nil
}

func RemoveDeniedPaths(root string, denylist []string) (int, error) {
	removed := 0
	for _, rel := range denylist {
		if rel == "" {
			continue
		}
		path := filepath.Join(root, rel)
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return removed, err
		}
		if err := os.RemoveAll(path); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func copyFileIfExists(src, dst string) (int, error) {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if info.IsDir() {
		return 0, fmt.Errorf("expected file but got dir: %s", src)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return 0, err
	}

	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return 0, err
	}
	if err := out.Close(); err != nil {
		return 0, err
	}

	return 1, os.Chmod(dst, info.Mode().Perm())
}

func copyDirIfExists(src, dst string) (int, error) {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("expected directory but got file: %s", src)
	}

	_ = os.RemoveAll(dst)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return 0, err
	}

	changed := 0
	err = filepath.Walk(src, func(path string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(os.PathSeparator)) {
			return nil
		}

		target := filepath.Join(dst, rel)
		if fi.IsDir() {
			return os.MkdirAll(target, fi.Mode().Perm())
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.Create(target)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
		if err := os.Chmod(target, fi.Mode().Perm()); err != nil {
			return err
		}
		changed++
		return nil
	})

	return changed, err
}

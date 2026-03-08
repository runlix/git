package sync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type CopyStats struct {
	SymlinkDirDereferenceCount   int
	SymlinkFileDereferenceCount  int
	SymlinkBrokenSkippedCount    int
	SymlinkOutOfRootSkippedCount int
	SymlinkCycleSkippedCount     int
}

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
		n, err := copyFileIfExists(src, dst, nil)
		if err != nil {
			return changed, err
		}
		changed += n
	}

	for _, rel := range dirs {
		src := filepath.Join(repoDir, rel)
		dst := filepath.Join(configDir, rel)
		n, err := copyDirIfExists(src, dst, nil)
		if err != nil {
			return changed, err
		}
		changed += n
	}

	return changed, nil
}

func CopyAllowlistedFromConfigToRepo(configDir, repoDir string, files, dirs []string) (int, error) {
	changed, _, err := CopyAllowlistedFromConfigToRepoWithStats(configDir, repoDir, files, dirs)
	return changed, err
}

func CopyAllowlistedFromConfigToRepoWithStats(configDir, repoDir string, files, dirs []string) (int, CopyStats, error) {
	changed := 0
	stats := CopyStats{}

	for _, rel := range files {
		src := filepath.Join(configDir, rel)
		dst := filepath.Join(repoDir, rel)
		n, err := copyFileIfExists(src, dst, &stats)
		if err != nil {
			return changed, stats, err
		}
		changed += n
	}

	for _, rel := range dirs {
		src := filepath.Join(configDir, rel)
		dst := filepath.Join(repoDir, rel)
		n, err := copyDirIfExists(src, dst, &stats)
		if err != nil {
			return changed, stats, err
		}
		changed += n
	}

	return changed, stats, nil
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

func copyFileIfExists(src, dst string, stats *CopyStats) (int, error) {
	info, err := os.Lstat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(src)
		if err != nil {
			if os.IsNotExist(err) {
				incrementStat(stats, "broken")
				return 0, nil
			}
			return 0, err
		}
		resolvedInfo, err := os.Stat(resolved)
		if err != nil {
			if os.IsNotExist(err) {
				incrementStat(stats, "broken")
				return 0, nil
			}
			return 0, err
		}
		if resolvedInfo.IsDir() {
			incrementStat(stats, "dir")
			return copyDirIfExists(resolved, dst, stats)
		}
		incrementStat(stats, "file")
		return copyFileIfExists(resolved, dst, stats)
	}
	if info.IsDir() {
		return copyDirIfExists(src, dst, stats)
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

func copyDirIfExists(src, dst string, stats *CopyStats) (int, error) {
	info, err := os.Lstat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(src)
		if err != nil {
			if os.IsNotExist(err) {
				incrementStat(stats, "broken")
				return 0, nil
			}
			return 0, err
		}
		resolvedAbs, err := filepath.Abs(resolved)
		if err != nil {
			return 0, err
		}
		srcAbs, err := filepath.Abs(src)
		if err != nil {
			return 0, err
		}
		srcRootReal, err := filepath.EvalSymlinks(srcAbs)
		if err != nil {
			return 0, err
		}
		if !withinRoot(srcRootReal, resolvedAbs) {
			incrementStat(stats, "out_of_root")
			return 0, nil
		}
		resolvedInfo, err := os.Stat(resolvedAbs)
		if err != nil {
			if os.IsNotExist(err) {
				incrementStat(stats, "broken")
				return 0, nil
			}
			return 0, err
		}
		if !resolvedInfo.IsDir() {
			incrementStat(stats, "file")
			return 0, nil
		}
		incrementStat(stats, "dir")
		return copyDirIfExists(resolvedAbs, dst, stats)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("expected directory but got file: %s", src)
	}

	_ = os.RemoveAll(dst)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return 0, err
	}

	srcAbs, err := filepath.Abs(src)
	if err != nil {
		return 0, err
	}
	srcRootReal, err := filepath.EvalSymlinks(srcAbs)
	if err != nil {
		return 0, err
	}

	changed := 0
	visited := map[string]int{}
	err = copyDirRecursive(src, dst, srcRootReal, "", visited, &changed, stats)

	return changed, err
}

func copyDirRecursive(srcDir, dstDir, srcRootReal, rel string, visited map[string]int, changed *int, stats *CopyStats) error {
	realDir, err := filepath.EvalSymlinks(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	realAbs, err := filepath.Abs(realDir)
	if err != nil {
		return err
	}
	if !withinRoot(srcRootReal, realAbs) {
		incrementStat(stats, "out_of_root")
		return nil
	}
	if visited[realAbs] > 0 {
		incrementStat(stats, "cycle")
		return nil
	}
	visited[realAbs]++
	defer func() { visited[realAbs]-- }()

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		childRel := filepath.ToSlash(filepath.Clean(filepath.Join(rel, name)))
		if childRel == ".git" || strings.HasPrefix(childRel, ".git/") {
			continue
		}

		srcPath := filepath.Join(srcDir, name)
		dstPath := filepath.Join(dstDir, name)

		info, err := os.Lstat(srcPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(srcPath)
			if err != nil {
				if os.IsNotExist(err) {
					incrementStat(stats, "broken")
					continue
				}
				return err
			}
			resolvedAbs, err := filepath.Abs(resolved)
			if err != nil {
				return err
			}
			if !withinRoot(srcRootReal, resolvedAbs) {
				incrementStat(stats, "out_of_root")
				continue
			}
			resolvedInfo, err := os.Stat(resolvedAbs)
			if err != nil {
				if os.IsNotExist(err) {
					incrementStat(stats, "broken")
					continue
				}
				return err
			}
			if resolvedInfo.IsDir() {
				incrementStat(stats, "dir")
				if err := os.RemoveAll(dstPath); err != nil {
					return err
				}
				if err := os.MkdirAll(dstPath, 0o755); err != nil {
					return err
				}
				if err := copyDirRecursive(resolvedAbs, dstPath, srcRootReal, childRel, visited, changed, stats); err != nil {
					return err
				}
				continue
			}
			incrementStat(stats, "file")
			if err := copyResolvedFile(resolvedAbs, dstPath, changed); err != nil {
				return err
			}
			continue
		}

		if info.IsDir() {
			if err := os.RemoveAll(dstPath); err != nil {
				return err
			}
			if err := os.MkdirAll(dstPath, info.Mode().Perm()); err != nil {
				return err
			}
			if err := copyDirRecursive(srcPath, dstPath, srcRootReal, childRel, visited, changed, stats); err != nil {
				return err
			}
			continue
		}

		if err := copyResolvedFile(srcPath, dstPath, changed); err != nil {
			return err
		}
	}

	return nil
}

func incrementStat(stats *CopyStats, key string) {
	if stats == nil {
		return
	}
	switch key {
	case "dir":
		stats.SymlinkDirDereferenceCount++
	case "file":
		stats.SymlinkFileDereferenceCount++
	case "broken":
		stats.SymlinkBrokenSkippedCount++
	case "out_of_root":
		stats.SymlinkOutOfRootSkippedCount++
	case "cycle":
		stats.SymlinkCycleSkippedCount++
	}
}

func copyResolvedFile(src, dst string, changed *int) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("expected file but got dir: %s", src)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if fi, err := os.Lstat(dst); err == nil {
		if fi.IsDir() {
			if err := os.RemoveAll(dst); err != nil {
				return err
			}
		} else if fi.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(dst); err != nil {
				return err
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
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
	if err := os.Chmod(dst, info.Mode().Perm()); err != nil {
		return err
	}
	(*changed)++
	return nil
}

func withinRoot(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, "../"))
}

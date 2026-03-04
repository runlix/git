package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/runlix/git/internal/githubapp"
	"github.com/runlix/git/internal/gitops"
	syncops "github.com/runlix/git/internal/sync"
)

var version = "dev"

type envConfig struct {
	repoURL                 string
	repoRef                 string
	workDir                 string
	configDir               string
	allowlistFiles          []string
	allowlistDirs           []string
	denylistPaths           []string
	gitAuthorName           string
	gitAuthorEmail          string
	commitMessageTemplate   string
	githubAppID             string
	githubAppInstallationID string
	githubAppPrivateKeyFile string
}

func main() {
	if len(os.Args) < 2 {
		fatal("missing subcommand; use pull-init, sync-push, or version", nil)
	}

	cmd := os.Args[1]
	if cmd == "version" {
		fmt.Println(version)
		return
	}

	cfg := loadEnvConfig()
	start := time.Now()

	token, err := githubapp.GetInstallationToken(
		cfg.githubAppID,
		cfg.githubAppInstallationID,
		cfg.githubAppPrivateKeyFile,
	)
	if err != nil {
		fatal("github app authentication failed", err)
	}

	authURL, err := githubapp.InjectTokenInHTTPSRepoURL(cfg.repoURL, token)
	if err != nil {
		fatal("failed to prepare authenticated repo url", err)
	}

	repoDir := filepath.Join(cfg.workDir, "repo")
	if err := gitops.PrepareRepo(repoDir, authURL, cfg.repoRef); err != nil {
		fatal("repository prepare failed", err)
	}

	switch cmd {
	case "pull-init":
		runPullInit(repoDir, cfg, start)
	case "sync-push":
		runSyncPush(repoDir, cfg, start)
	default:
		fatal("unknown subcommand", errors.New(cmd))
	}
}

func runPullInit(repoDir string, cfg envConfig, start time.Time) {
	changed, err := syncops.CopyAllowlistedFromRepoToConfig(
		repoDir,
		cfg.configDir,
		cfg.allowlistFiles,
		cfg.allowlistDirs,
	)
	if err != nil {
		fatal("pull-init copy failed", err)
	}

	_ = os.RemoveAll(filepath.Join(cfg.configDir, ".git"))

	logKV(
		"level", "info",
		"operation", "pull-init",
		"repo", sanitizeRepo(cfg.repoURL),
		"ref", cfg.repoRef,
		"changed_paths_count", fmt.Sprintf("%d", changed),
		"duration_ms", fmt.Sprintf("%d", time.Since(start).Milliseconds()),
	)
}

func runSyncPush(repoDir string, cfg envConfig, start time.Time) {
	changed, err := syncops.CopyAllowlistedFromConfigToRepo(
		cfg.configDir,
		repoDir,
		cfg.allowlistFiles,
		cfg.allowlistDirs,
	)
	if err != nil {
		fatal("sync-push copy failed", err)
	}

	if len(cfg.denylistPaths) > 0 {
		removed, err := syncops.RemoveDeniedPaths(repoDir, cfg.denylistPaths)
		if err != nil {
			fatal("sync-push denylist enforcement failed", err)
		}
		changed += removed
	}

	status, err := gitops.StatusPorcelain(repoDir)
	if err != nil {
		fatal("sync-push status check failed", err)
	}
	if strings.TrimSpace(status) == "" {
		logKV(
			"level", "info",
			"operation", "sync-push",
			"repo", sanitizeRepo(cfg.repoURL),
			"ref", cfg.repoRef,
			"changed_paths_count", "0",
			"duration_ms", fmt.Sprintf("%d", time.Since(start).Milliseconds()),
		)
		return
	}

	if err := gitops.ConfigureAuthor(repoDir, cfg.gitAuthorName, cfg.gitAuthorEmail); err != nil {
		fatal("sync-push author config failed", err)
	}

	if err := gitops.AddAll(repoDir); err != nil {
		fatal("sync-push add failed", err)
	}

	message := strings.ReplaceAll(cfg.commitMessageTemplate, "{{ref}}", cfg.repoRef)
	message = strings.ReplaceAll(message, "{{timestamp}}", time.Now().UTC().Format(time.RFC3339))
	if err := gitops.Commit(repoDir, message); err != nil {
		fatal("sync-push commit failed", err)
	}

	sha, err := gitops.RevParseHead(repoDir)
	if err != nil {
		fatal("sync-push rev-parse failed", err)
	}
	if err := gitops.Push(repoDir, "origin", "HEAD:"+cfg.repoRef); err != nil {
		fatal("sync-push push failed", err)
	}

	logKV(
		"level", "info",
		"operation", "sync-push",
		"repo", sanitizeRepo(cfg.repoURL),
		"ref", cfg.repoRef,
		"changed_paths_count", fmt.Sprintf("%d", changed),
		"commit_sha", sha,
		"duration_ms", fmt.Sprintf("%d", time.Since(start).Milliseconds()),
	)
}

func loadEnvConfig() envConfig {
	cfg := envConfig{
		repoURL:                 mustEnv("REPO_URL"),
		repoRef:                 envOrDefault("REPO_REF", "main"),
		workDir:                 envOrDefault("WORK_DIR", "/work"),
		configDir:               envOrDefault("CONFIG_DIR", "/config"),
		allowlistFiles:          syncops.ParseCSV(envOrDefault("ALLOWLIST_FILES", "configuration.yaml,automations.yaml,scripts.yaml,scenes.yaml")),
		allowlistDirs:           syncops.ParseCSV(envOrDefault("ALLOWLIST_DIRS", "packages,themes,www,blueprints,custom_components")),
		denylistPaths:           syncops.ParseCSV(envOrDefault("DENYLIST_PATHS", "secrets.yaml,.storage,home-assistant_v2.db,home-assistant.log,deps,tts,cloud,ssh")),
		gitAuthorName:           envOrDefault("GIT_AUTHOR_NAME", "runlix-gitops"),
		gitAuthorEmail:          envOrDefault("GIT_AUTHOR_EMAIL", "gitops@runlix.local"),
		commitMessageTemplate:   envOrDefault("COMMIT_MESSAGE_TEMPLATE", "Home Assistant config sync ({{ref}} @ {{timestamp}})"),
		githubAppID:             mustEnv("GITHUB_APP_ID"),
		githubAppInstallationID: mustEnv("GITHUB_APP_INSTALLATION_ID"),
		githubAppPrivateKeyFile: mustEnv("GITHUB_APP_PRIVATE_KEY_FILE"),
	}

	_ = os.MkdirAll(cfg.workDir, 0o755)
	_ = os.MkdirAll(cfg.configDir, 0o755)

	return cfg
}

func logKV(fields ...string) {
	pairs := make([]string, 0, len(fields)/2)
	for i := 0; i+1 < len(fields); i += 2 {
		k := strings.ReplaceAll(strings.TrimSpace(fields[i]), " ", "_")
		v := strings.TrimSpace(fields[i+1])
		pairs = append(pairs, fmt.Sprintf("%s=%q", k, v))
	}
	fmt.Println(strings.Join(pairs, " "))
}

func sanitizeRepo(repoURL string) string {
	u, err := githubapp.ParseRepoURL(repoURL)
	if err != nil {
		return "invalid-url"
	}
	if u.Path == "" {
		return u.Host
	}
	return u.Host + u.Path
}

func mustEnv(name string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		fatal("missing required environment variable", errors.New(name))
	}
	return v
}

func envOrDefault(name, defaultValue string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return defaultValue
	}
	return v
}

func fatal(message string, err error) {
	if err == nil {
		logKV("level", "error", "msg", message)
		os.Exit(1)
	}
	logKV("level", "error", "msg", message, "error", err.Error())
	os.Exit(1)
}

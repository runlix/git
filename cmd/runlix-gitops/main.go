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

type errorCode string

const (
	errorCodeConfigMissing     errorCode = "CONFIG_MISSING"
	errorCodeAuthGitHubApp     errorCode = "AUTH_GITHUB_APP"
	errorCodeRepoPrepare       errorCode = "REPO_PREPARE"
	errorCodeCopyAllowlist     errorCode = "COPY_ALLOWLIST"
	errorCodeDenylistEnforce   errorCode = "DENYLIST_ENFORCE"
	errorCodeGitStatus         errorCode = "GIT_STATUS"
	errorCodeGitCommit         errorCode = "GIT_COMMIT"
	errorCodeGitPush           errorCode = "GIT_PUSH"
	errorCodeUnknownSubcommand errorCode = "UNKNOWN_SUBCOMMAND"
)

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
	start := time.Now()

	if len(os.Args) < 2 {
		fatalWith(errorCodeUnknownSubcommand, "startup", "missing subcommand; use pull-init, sync-push, or version", nil, start)
	}

	cmd := os.Args[1]
	if cmd == "version" {
		fmt.Println(version)
		return
	}

	cfg, err := loadEnvConfig()
	if err != nil {
		fatalWith(errorCodeConfigMissing, cmd, "invalid environment configuration", err, start)
	}

	token, err := githubapp.GetInstallationToken(
		cfg.githubAppID,
		cfg.githubAppInstallationID,
		cfg.githubAppPrivateKeyFile,
	)
	if err != nil {
		fatalWith(errorCodeAuthGitHubApp, cmd, "github app authentication failed", err, start)
	}

	authURL, err := githubapp.InjectTokenInHTTPSRepoURL(cfg.repoURL, token)
	if err != nil {
		fatalWith(errorCodeConfigMissing, cmd, "failed to prepare authenticated repo url", err, start)
	}

	repoDir := filepath.Join(cfg.workDir, "repo")
	if err := gitops.PrepareRepo(repoDir, authURL, cfg.repoRef); err != nil {
		fatalWith(errorCodeRepoPrepare, cmd, "repository prepare failed", err, start)
	}

	switch cmd {
	case "pull-init":
		runPullInit(repoDir, cfg, start)
	case "sync-push":
		runSyncPush(repoDir, cfg, start)
	default:
		fatalWith(errorCodeUnknownSubcommand, cmd, "unknown subcommand", errors.New(cmd), start)
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
		fatalWith(errorCodeCopyAllowlist, "pull-init", "pull-init copy failed", err, start)
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
		fatalWith(errorCodeCopyAllowlist, "sync-push", "sync-push copy failed", err, start)
	}

	if len(cfg.denylistPaths) > 0 {
		removed, err := syncops.RemoveDeniedPaths(repoDir, cfg.denylistPaths)
		if err != nil {
			fatalWith(errorCodeDenylistEnforce, "sync-push", "sync-push denylist enforcement failed", err, start)
		}
		changed += removed
	}

	status, err := gitops.StatusPorcelain(repoDir)
	if err != nil {
		fatalWith(errorCodeGitStatus, "sync-push", "sync-push status check failed", err, start)
	}
	if !hasRepoChanges(status) {
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
		fatalWith(errorCodeGitCommit, "sync-push", "sync-push author config failed", err, start)
	}

	if err := gitops.AddAll(repoDir); err != nil {
		fatalWith(errorCodeGitCommit, "sync-push", "sync-push add failed", err, start)
	}

	message := strings.ReplaceAll(cfg.commitMessageTemplate, "{{ref}}", cfg.repoRef)
	message = strings.ReplaceAll(message, "{{timestamp}}", time.Now().UTC().Format(time.RFC3339))
	if err := gitops.Commit(repoDir, message); err != nil {
		fatalWith(errorCodeGitCommit, "sync-push", "sync-push commit failed", err, start)
	}

	sha, err := gitops.RevParseHead(repoDir)
	if err != nil {
		fatalWith(errorCodeGitCommit, "sync-push", "sync-push rev-parse failed", err, start)
	}
	if err := gitops.Push(repoDir, "origin", "HEAD:"+cfg.repoRef); err != nil {
		fatalWith(errorCodeGitPush, "sync-push", "sync-push push failed", err, start)
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

func loadEnvConfig() (envConfig, error) {
	repoURL, err := mustEnv("REPO_URL")
	if err != nil {
		return envConfig{}, err
	}
	appID, err := mustEnv("GITHUB_APP_ID")
	if err != nil {
		return envConfig{}, err
	}
	installationID, err := mustEnv("GITHUB_APP_INSTALLATION_ID")
	if err != nil {
		return envConfig{}, err
	}
	privateKeyFile, err := mustEnv("GITHUB_APP_PRIVATE_KEY_FILE")
	if err != nil {
		return envConfig{}, err
	}

	cfg := envConfig{
		repoURL:                 repoURL,
		repoRef:                 envOrDefault("REPO_REF", "main"),
		workDir:                 envOrDefault("WORK_DIR", "/work"),
		configDir:               envOrDefault("CONFIG_DIR", "/config"),
		allowlistFiles:          syncops.ParseCSV(envOrDefault("ALLOWLIST_FILES", "configuration.yaml,automations.yaml,scripts.yaml,scenes.yaml")),
		allowlistDirs:           syncops.ParseCSV(envOrDefault("ALLOWLIST_DIRS", "packages,themes,www,blueprints,custom_components")),
		denylistPaths:           syncops.ParseCSV(envOrDefault("DENYLIST_PATHS", "secrets.yaml,.storage,home-assistant_v2.db,home-assistant.log,deps,tts,cloud,ssh")),
		gitAuthorName:           envOrDefault("GIT_AUTHOR_NAME", "runlix-gitops"),
		gitAuthorEmail:          envOrDefault("GIT_AUTHOR_EMAIL", "gitops@runlix.local"),
		commitMessageTemplate:   envOrDefault("COMMIT_MESSAGE_TEMPLATE", "Home Assistant config sync ({{ref}} @ {{timestamp}})"),
		githubAppID:             appID,
		githubAppInstallationID: installationID,
		githubAppPrivateKeyFile: privateKeyFile,
	}

	_ = os.MkdirAll(cfg.workDir, 0o755)
	_ = os.MkdirAll(cfg.configDir, 0o755)

	return cfg, nil
}

func hasRepoChanges(status string) bool {
	return strings.TrimSpace(status) != ""
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

func mustEnv(name string) (string, error) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return "", fmt.Errorf("missing required environment variable: %s", name)
	}
	return v, nil
}

func envOrDefault(name, defaultValue string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return defaultValue
	}
	return v
}

func fatalWith(code errorCode, operation, message string, err error, start time.Time) {
	fields := []string{
		"level", "error",
		"operation", operation,
		"error_code", string(code),
		"msg", message,
		"duration_ms", fmt.Sprintf("%d", time.Since(start).Milliseconds()),
	}
	if err != nil {
		fields = append(fields, "error", gitops.SanitizeText(err.Error()))
	}
	logKV(fields...)
	os.Exit(1)
}

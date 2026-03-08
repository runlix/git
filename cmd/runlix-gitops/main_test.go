package main

import (
	"testing"

	syncops "github.com/runlix/git/internal/sync"
)

func TestHasRepoChanges(t *testing.T) {
	if hasRepoChanges("\n") {
		t.Fatal("expected false for empty-like status")
	}
	if !hasRepoChanges(" M configuration.yaml") {
		t.Fatal("expected true for modified status")
	}
}

func TestEnvOrDefault(t *testing.T) {
	t.Setenv("UNIT_TEST_ENV", "")
	if got := envOrDefault("UNIT_TEST_ENV", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
	t.Setenv("UNIT_TEST_ENV", "value")
	if got := envOrDefault("UNIT_TEST_ENV", "fallback"); got != "value" {
		t.Fatalf("expected value, got %q", got)
	}
}

func TestMustEnv(t *testing.T) {
	t.Setenv("UNIT_MUST_ENV", "abc")
	if got, err := mustEnv("UNIT_MUST_ENV"); err != nil || got != "abc" {
		t.Fatalf("expected abc,nil got %q,%v", got, err)
	}
	t.Setenv("UNIT_MUST_ENV", "")
	if _, err := mustEnv("UNIT_MUST_ENV"); err == nil {
		t.Fatal("expected error for missing env")
	}
}

func TestSymlinkStatsFields(t *testing.T) {
	fields := symlinkStatsFields(syncops.CopyStats{
		SymlinkDirDereferenceCount:   2,
		SymlinkFileDereferenceCount:  1,
		SymlinkBrokenSkippedCount:    3,
		SymlinkOutOfRootSkippedCount: 4,
		SymlinkCycleSkippedCount:     5,
	})

	want := map[string]string{
		"symlink_policy":                    "dereference_copy",
		"symlink_dir_deref_count":           "2",
		"symlink_file_deref_count":          "1",
		"symlink_broken_skipped_count":      "3",
		"symlink_out_of_root_skipped_count": "4",
		"symlink_cycle_skipped_count":       "5",
	}

	if len(fields)%2 != 0 {
		t.Fatalf("expected key/value pairs, got odd length: %d", len(fields))
	}

	got := map[string]string{}
	for i := 0; i+1 < len(fields); i += 2 {
		got[fields[i]] = fields[i+1]
	}

	for k, v := range want {
		if got[k] != v {
			t.Fatalf("unexpected field %s: got=%q want=%q", k, got[k], v)
		}
	}
}

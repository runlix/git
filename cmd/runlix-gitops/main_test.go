package main

import "testing"

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

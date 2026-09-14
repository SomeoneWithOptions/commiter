package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeTestConfig(t *testing.T, path, contents string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), mode); err != nil {
		t.Fatal(err)
	}
}

func TestResolveOpenRouterAPIKeyPrefersConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeTestConfig(t, path, `{"openrouter":{"api_key":"from-config"}}`, 0o600)
	t.Setenv("OPENROUTER_API_KEY", "from-env")

	got, err := resolveOpenRouterAPIKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "from-config" {
		t.Fatalf("got %q, want config key", got)
	}
}

func TestResolveOpenRouterAPIKeyFallsBackToEnvironment(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("OPENROUTER_API_KEY", "from-env")

	got, err := resolveOpenRouterAPIKey("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "from-env" {
		t.Fatalf("got %q, want environment key", got)
	}
}

func TestResolveOpenRouterAPIKeyUsesDefaultConfig(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("OPENROUTER_API_KEY", "from-env")
	path := filepath.Join(configDir, "commiter", "config.json")
	writeTestConfig(t, path, `{"openrouter":{"api_key":"from-default-config"}}`, 0o600)

	got, err := resolveOpenRouterAPIKey("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "from-default-config" {
		t.Fatalf("got %q, want default config key", got)
	}
}

func TestResolveOpenRouterAPIKeyRejectsOpenPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not enforced on Windows")
	}

	path := filepath.Join(t.TempDir(), "config.json")
	writeTestConfig(t, path, `{"openrouter":{"api_key":"secret"}}`, 0o644)
	t.Setenv("OPENROUTER_API_KEY", "from-env")

	_, err := resolveOpenRouterAPIKey(path)
	if err == nil || !strings.Contains(err.Error(), "permissions are too open") {
		t.Fatalf("got error %v, want permissions error", err)
	}
}

func TestResolveOpenRouterAPIKeyRejectsMalformedConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeTestConfig(t, path, `{"openrouter":`, 0o600)

	_, err := resolveOpenRouterAPIKey(path)
	if err == nil || !strings.Contains(err.Error(), "failed to parse config") {
		t.Fatalf("got error %v, want parse error", err)
	}
}

func TestResolveOpenRouterAPIKeyReportsMissingCredential(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("OPENROUTER_API_KEY", "")

	_, err := resolveOpenRouterAPIKey("")
	if err == nil || !strings.Contains(err.Error(), "OpenRouter API key not found") {
		t.Fatalf("got error %v, want missing credential error", err)
	}
}

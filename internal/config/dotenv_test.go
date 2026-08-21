package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDotEnvLine(t *testing.T) {
	cases := []struct {
		line      string
		wantKey   string
		wantValue string
		wantOK    bool
	}{
		{"TG_ENABLED=true", "TG_ENABLED", "true", true},
		{"export TG_CHAT_ID=-100123", "TG_CHAT_ID", "-100123", true},
		{"TG_BOT_TOKEN=\"abc def\"", "TG_BOT_TOKEN", "abc def", true},
		{"# comment", "", "", false},
		{"", "", "", false},
		{"NOVALUE", "", "", false},
	}

	for _, tc := range cases {
		key, value, ok := parseDotEnvLine(tc.line)
		if ok != tc.wantOK || key != tc.wantKey || value != tc.wantValue {
			t.Fatalf("parseDotEnvLine(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tc.line, key, value, ok, tc.wantKey, tc.wantValue, tc.wantOK)
		}
	}
}

func TestApplyDotEnvFileDoesNotOverrideExistingEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("CFG_TEST_KEY=from-file\nCFG_TEST_KEEP=file-value\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	t.Setenv("CFG_TEST_KEEP", "shell-value")
	_ = os.Unsetenv("CFG_TEST_KEY")

	if err := applyDotEnvFile(path); err != nil {
		t.Fatalf("applyDotEnvFile: %v", err)
	}

	if got := os.Getenv("CFG_TEST_KEY"); got != "from-file" {
		t.Fatalf("CFG_TEST_KEY=%q, want from-file", got)
	}
	if got := os.Getenv("CFG_TEST_KEEP"); got != "shell-value" {
		t.Fatalf("CFG_TEST_KEEP=%q, want shell-value", got)
	}
}

func TestLoadReadsDotEnvFromWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("APP_PORT=9876\nTG_ENABLED=false\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	_ = os.Unsetenv("APP_PORT")
	_ = os.Unsetenv("TG_ENABLED")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AppPort != "9876" {
		t.Fatalf("AppPort=%q, want 9876", cfg.AppPort)
	}
}

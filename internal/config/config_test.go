package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadParsesEnvWithDefaultsAndQuotes(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".env")
	content := `
# comment
APP_LOGIN_USER="admin"
APP_LOGIN_KEY='secret-key'
APP_SESSION_SECRET=0123456789abcdef
PG_HOST=127.0.0.1
PG_SUPERUSER=postgres
PG_SUPER_PASSWORD='postgres password'
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AppAddr != "127.0.0.1:8080" {
		t.Fatalf("AppAddr = %q", cfg.AppAddr)
	}
	if cfg.AppLoginUser != "admin" || cfg.AppLoginKey != "secret-key" {
		t.Fatalf("login config not parsed: %#v", cfg)
	}
	if cfg.Postgres.Port != "5432" || cfg.Postgres.AdminDB != "postgres" || cfg.Postgres.SSLMode != "disable" {
		t.Fatalf("postgres defaults not applied: %#v", cfg.Postgres)
	}
	if cfg.Postgres.SuperPassword != "postgres password" {
		t.Fatalf("quoted password not trimmed: %q", cfg.Postgres.SuperPassword)
	}
}

func TestLoadReportsMissingRequiredValues(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("APP_LOGIN_USER=admin\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() expected error")
	}
	for _, key := range []string{"APP_LOGIN_KEY", "APP_SESSION_SECRET", "PG_HOST", "PG_SUPERUSER", "PG_SUPER_PASSWORD"} {
		if !strings.Contains(err.Error(), key) {
			t.Fatalf("error %q does not mention %s", err.Error(), key)
		}
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".env")
	content := `
APP_LOGIN_USER=admin
APP_LOGIN_KEY=secret
APP_SESSION_SECRET=0123456789abcdef
PG_HOST=127.0.0.1
PG_PORT=not-number
PG_SUPERUSER=postgres
PG_SUPER_PASSWORD=postgres
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "PG_PORT") {
		t.Fatalf("Load() error = %v, want PG_PORT error", err)
	}
}

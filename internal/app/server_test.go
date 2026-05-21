package app

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"pg_provision_dbuser_codex/internal/config"
	"pg_provision_dbuser_codex/internal/session"
)

func testConfig() config.Config {
	return config.Config{
		AppAddr:          "127.0.0.1:8080",
		AppLoginUser:     "admin",
		AppLoginKey:      "login-key",
		AppSessionSecret: "0123456789abcdef",
		Postgres: config.PostgresConfig{
			Host:          "127.0.0.1",
			Port:          "5432",
			AdminDB:       "postgres",
			SuperUser:     "postgres",
			SuperPassword: "postgres",
			SSLMode:       "disable",
		},
	}
}

func TestUnauthenticatedHomeRedirectsToLogin(t *testing.T) {
	t.Parallel()

	server, err := New(testConfig())
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	server.Routes().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusSeeOther)
	}
	if location := recorder.Header().Get("Location"); location != "/login" {
		t.Fatalf("Location = %q, want /login", location)
	}
}

func TestLoginFailureAndSuccess(t *testing.T) {
	t.Parallel()

	server, err := New(testConfig())
	if err != nil {
		t.Fatal(err)
	}

	badForm := url.Values{"username": {"admin"}, "login_key": {"wrong"}}
	badRecorder := httptest.NewRecorder()
	badRequest := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(badForm.Encode()))
	badRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.Routes().ServeHTTP(badRecorder, badRequest)
	if badRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("bad login status = %d, want %d", badRecorder.Code, http.StatusUnauthorized)
	}

	goodForm := url.Values{"username": {"admin"}, "login_key": {"login-key"}}
	goodRecorder := httptest.NewRecorder()
	goodRequest := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(goodForm.Encode()))
	goodRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	server.Routes().ServeHTTP(goodRecorder, goodRequest)
	if goodRecorder.Code != http.StatusSeeOther {
		t.Fatalf("good login status = %d, want %d", goodRecorder.Code, http.StatusSeeOther)
	}
	if location := goodRecorder.Header().Get("Location"); location != "/" {
		t.Fatalf("Location = %q, want /", location)
	}
	if cookie := findCookie(goodRecorder.Result().Cookies(), session.CookieName); cookie == nil || cookie.Value == "" {
		t.Fatalf("login did not set %s cookie", session.CookieName)
	}
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

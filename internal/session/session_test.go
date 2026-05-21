package session

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestManagerIssuesAndValidatesCookie(t *testing.T) {
	t.Parallel()

	manager := NewManager("0123456789abcdef")
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }

	recorder := httptest.NewRecorder()
	manager.Issue(recorder, "admin")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range recorder.Result().Cookies() {
		req.AddCookie(cookie)
	}

	username, ok := manager.Username(req)
	if !ok || username != "admin" {
		t.Fatalf("Username() = %q, %v", username, ok)
	}
}

func TestManagerRejectsExpiredCookie(t *testing.T) {
	t.Parallel()

	manager := NewManager("0123456789abcdef")
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }

	recorder := httptest.NewRecorder()
	manager.Issue(recorder, "admin")

	manager.now = func() time.Time { return now.Add(9 * time.Hour) }
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range recorder.Result().Cookies() {
		req.AddCookie(cookie)
	}

	if username, ok := manager.Username(req); ok || username != "" {
		t.Fatalf("Username() = %q, %v, want expired", username, ok)
	}
}

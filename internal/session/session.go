package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const CookieName = "pg_provision_session"

type Manager struct {
	secret []byte
	now    func() time.Time
	ttl    time.Duration
}

func NewManager(secret string) *Manager {
	return &Manager{
		secret: []byte(secret),
		now:    time.Now,
		ttl:    8 * time.Hour,
	}
}

func (m *Manager) Issue(w http.ResponseWriter, username string) {
	expiresAt := m.now().Add(m.ttl)
	payload := fmt.Sprintf("%s|%d", username, expiresAt.Unix())
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    m.sign(payload),
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (m *Manager) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (m *Manager) Username(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(CookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}

	payload, ok := m.verify(cookie.Value)
	if !ok {
		return "", false
	}

	username, expiresText, ok := strings.Cut(payload, "|")
	if !ok || username == "" {
		return "", false
	}
	expiresUnix, err := strconv.ParseInt(expiresText, 10, 64)
	if err != nil {
		return "", false
	}
	if !m.now().Before(time.Unix(expiresUnix, 0)) {
		return "", false
	}
	return username, true
}

func (m *Manager) sign(payload string) string {
	payloadText := base64.RawURLEncoding.EncodeToString([]byte(payload))
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(payloadText))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payloadText + "." + signature
}

func (m *Manager) verify(value string) (string, bool) {
	payloadText, signature, ok := strings.Cut(value, ".")
	if !ok || payloadText == "" || signature == "" {
		return "", false
	}

	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(payloadText))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return "", false
	}

	payload, err := base64.RawURLEncoding.DecodeString(payloadText)
	if err != nil {
		return "", false
	}
	return string(payload), true
}

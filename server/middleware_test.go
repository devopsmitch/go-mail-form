package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devopsmitch/go-mail-form/config"
)

func TestResolveRedirectAbsolute(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	got := resolveRedirectURL(r, "https://example.com/thanks")
	if got != "https://example.com/thanks" {
		t.Fatalf("expected absolute URL unchanged, got %s", got)
	}
}

func TestResolveRedirectRelative(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	r.Header.Set("Referer", "https://example.com/contact")
	got := resolveRedirectURL(r, "/thanks")
	if got != "https://example.com/thanks" {
		t.Fatalf("expected https://example.com/thanks, got %s", got)
	}
}

func TestResolveRedirectNoReferer(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	got := resolveRedirectURL(r, "/thanks")
	if got != "/thanks" {
		t.Fatalf("expected /thanks as fallback, got %s", got)
	}
}

func TestRedirectOrJSONWithRedirect(t *testing.T) {
	target := &config.Target{
		SMTP:       "smtps://user:pass@smtp.example.com",
		Recipients: []string{"to@example.com"},
		RateLimit:  &config.RateLimit{Timespan: 60, Requests: 10},
		Redirect: &config.Redirect{
			Success: "https://example.com/thanks",
			Error:   "https://example.com/error",
		},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	redirectOrJSON(target, w, r, http.StatusOK, "ok")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "https://example.com/thanks" {
		t.Fatalf("expected success redirect, got %s", loc)
	}

	w = httptest.NewRecorder()
	redirectOrJSON(target, w, r, http.StatusInternalServerError, "fail")
	if loc := w.Header().Get("Location"); loc != "https://example.com/error" {
		t.Fatalf("expected error redirect, got %s", loc)
	}
}

func TestRedirectOrJSONWithoutRedirect(t *testing.T) {
	target := &config.Target{
		SMTP:       "smtps://user:pass@smtp.example.com",
		Recipients: []string{"to@example.com"},
		RateLimit:  &config.RateLimit{Timespan: 60, Requests: 10},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/test", nil)
	redirectOrJSON(target, w, r, http.StatusForbidden, "origin not allowed")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected JSON content-type, got %q", ct)
	}
}

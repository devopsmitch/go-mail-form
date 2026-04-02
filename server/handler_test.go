package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/devopsmitch/go-mail-form/config"
	"github.com/devopsmitch/go-mail-form/mail"
)

func testServer(sender MailSender) *Server {
	targets := map[string]*config.Target{
		"test": {
			SMTP:       "smtps://user:pass@smtp.example.com",
			Recipients: []string{"to@example.com"},
			From:       "noreply@example.com",
			RateLimit:  &config.RateLimit{Timespan: 60, Requests: 10},
		},
		"no-from": {
			SMTP:       "smtps://user:pass@smtp.example.com",
			Recipients: []string{"to@example.com"},
			RateLimit:  &config.RateLimit{Timespan: 60, Requests: 10},
		},
		"with-key": {
			SMTP:       "smtps://user:pass@smtp.example.com",
			Recipients: []string{"to@example.com"},
			From:       "noreply@example.com",
			Key:        "secret123",
			RateLimit:  &config.RateLimit{Timespan: 60, Requests: 10},
		},
		"with-origin": {
			SMTP:       "smtps://user:pass@smtp.example.com",
			Recipients: []string{"to@example.com"},
			From:       "noreply@example.com",
			Origin:     "https://example.com",
			RateLimit:  &config.RateLimit{Timespan: 60, Requests: 10},
		},
	}
	return New(targets, sender)
}

func noopSender() MailSender {
	return MailSenderFunc(func(ctx context.Context, target *config.Target, from, replyTo, subject, body string, attachments []mail.Attachment) error {
		return nil
	})
}

func failSender() MailSender {
	return MailSenderFunc(func(ctx context.Context, target *config.Target, from, replyTo, subject, body string, attachments []mail.Attachment) error {
		return errors.New("smtp error")
	})
}

func postForm(path string, data url.Values) *http.Request {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(data.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func validForm() url.Values {
	return url.Values{
		"from":    {"user@example.com"},
		"subject": {"Hello"},
		"body":    {"This is a test message"},
	}
}

func assertCode(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Fatalf("expected %d, got %d (body: %s)", want, w.Code, w.Body.String())
	}
}

func assertJSON(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected JSON content-type, got %q", ct)
	}
}

func TestHealthCheck(t *testing.T) {
	srv := testServer(noopSender())
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	assertCode(t, w, http.StatusOK)
}

func TestHealthCheckMethodNotAllowed(t *testing.T) {
	srv := testServer(noopSender())
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/healthz", nil))
	assertCode(t, w, http.StatusMethodNotAllowed)
}

func TestTargetNotFound(t *testing.T) {
	srv := testServer(noopSender())
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, postForm("/nonexistent", validForm()))
	assertCode(t, w, http.StatusNotFound)
	assertJSON(t, w)
}

func TestPathTraversalRejected(t *testing.T) {
	srv := testServer(noopSender())
	// ServeMux redirects /../ paths before our handler sees them
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, postForm("/../../etc", validForm()))
	if w.Code != http.StatusMovedPermanently && w.Code != http.StatusSeeOther && w.Code != 307 {
		t.Fatalf("expected redirect, got %d", w.Code)
	}

	// Slashes in target name are rejected by our handler
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, postForm("/foo/bar", validForm()))
	assertCode(t, w, http.StatusNotFound)
}

func TestMethodNotAllowed(t *testing.T) {
	srv := testServer(noopSender())
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/test", nil))
	assertCode(t, w, http.StatusMethodNotAllowed)
	assertJSON(t, w)
}

func TestSuccessfulSend(t *testing.T) {
	srv := testServer(noopSender())
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, postForm("/test", validForm()))
	assertCode(t, w, http.StatusOK)
}

func TestSendFailure(t *testing.T) {
	srv := testServer(failSender())
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, postForm("/test", validForm()))
	assertCode(t, w, http.StatusInternalServerError)
	assertJSON(t, w)
}

func TestHoneypotRejects(t *testing.T) {
	srv := testServer(failSender()) // would fail if actually called
	form := validForm()
	form.Set("_gotcha", "bot-filled-this")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, postForm("/test", form))
	assertCode(t, w, http.StatusOK)
}

func TestEmptyFromWhenNoDefault(t *testing.T) {
	srv := testServer(noopSender())
	form := url.Values{
		"subject": {"Hello"},
		"body":    {"This is a test message"},
	}
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, postForm("/no-from", form))
	assertCode(t, w, http.StatusUnprocessableEntity)
	assertJSON(t, w)
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name    string
		form    url.Values
		wantErr string
	}{
		{"missing subject", url.Values{"body": {"hello world"}}, "subject"},
		{"missing body", url.Values{"subject": {"Hi"}}, "body"},
		{"short subject", url.Values{"subject": {"H"}, "body": {"hello world"}}, "subject"},
		{"short body", url.Values{"subject": {"Hello"}, "body": {"hi"}}, "body"},
		{"bad email", url.Values{"from": {"not-an-email"}, "subject": {"Hello"}, "body": {"hello world"}}, "from"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := testServer(noopSender())
			w := httptest.NewRecorder()
			srv.Handler().ServeHTTP(w, postForm("/test", tt.form))
			assertCode(t, w, http.StatusUnprocessableEntity)
			assertJSON(t, w)
			var resp map[string]any
			json.NewDecoder(w.Body).Decode(&resp)
			problems := resp["problems"].([]any)
			found := false
			for _, p := range problems {
				if p.(map[string]any)["field"] == tt.wantErr {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected problem for field %q, got %v", tt.wantErr, resp)
			}
		})
	}
}

func TestAuthRequired(t *testing.T) {
	srv := testServer(noopSender())

	// No auth header
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, postForm("/with-key", validForm()))
	assertCode(t, w, http.StatusUnauthorized)
	assertJSON(t, w)

	// Wrong key
	w = httptest.NewRecorder()
	r := postForm("/with-key", validForm())
	r.Header.Set("Authorization", "Bearer wrong")
	srv.Handler().ServeHTTP(w, r)
	assertCode(t, w, http.StatusUnauthorized)

	// Correct key
	w = httptest.NewRecorder()
	r = postForm("/with-key", validForm())
	r.Header.Set("Authorization", "Bearer secret123")
	srv.Handler().ServeHTTP(w, r)
	assertCode(t, w, http.StatusOK)
}

func TestOriginCheck(t *testing.T) {
	srv := testServer(noopSender())

	// Wrong origin
	w := httptest.NewRecorder()
	r := postForm("/with-origin", validForm())
	r.Header.Set("Origin", "https://evil.com")
	srv.Handler().ServeHTTP(w, r)
	assertCode(t, w, http.StatusForbidden)
	assertJSON(t, w)

	// Correct origin
	w = httptest.NewRecorder()
	r = postForm("/with-origin", validForm())
	r.Header.Set("Origin", "https://example.com")
	srv.Handler().ServeHTTP(w, r)
	assertCode(t, w, http.StatusOK)
}

func TestCORSPreflight(t *testing.T) {
	srv := testServer(noopSender())
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodOptions, "/test", nil)
	srv.Handler().ServeHTTP(w, r)
	assertCode(t, w, http.StatusOK)
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("expected CORS allow origin *")
	}
}

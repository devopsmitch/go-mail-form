package antispam

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTurnstileClient(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantOK   bool
		wantErr  bool
	}{
		{"success", `{"success":true}`, true, false},
		{"failure", `{"success":false}`, false, false},
		{"bad json", `not json`, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.FormValue("secret") != "test-secret" {
					t.Errorf("expected secret=test-secret, got %q", r.FormValue("secret"))
				}
				if r.FormValue("response") != "test-token" {
					t.Errorf("expected response=test-token, got %q", r.FormValue("response"))
				}
				if r.FormValue("remoteip") != "1.2.3.4" {
					t.Errorf("expected remoteip=1.2.3.4, got %q", r.FormValue("remoteip"))
				}
				fmt.Fprint(w, tt.response)
			}))
			defer srv.Close()

			orig := turnstileVerifyURL
			defer func() { turnstileVerifyURL = orig }()
			turnstileVerifyURL = srv.URL

			c := &TurnstileClient{Secret: "test-secret"}
			ok, err := c.Verify(context.Background(), "test-token", "1.2.3.4")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

package server

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/devopsmitch/go-mail-form/config"
)

func cors(target *config.Target, w http.ResponseWriter, r *http.Request) bool {
	origin := "*"
	if target.Origin != "" {
		origin = target.Origin
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return true
	}
	return false
}

func checkOrigin(target *config.Target, r *http.Request) bool {
	if target.Origin == "" {
		return true
	}
	return r.Header.Get("Origin") == target.Origin
}

func checkAuth(target *config.Target, r *http.Request) bool {
	if target.Key == "" {
		return true
	}
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return false
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	return subtle.ConstantTimeCompare([]byte(token), []byte(target.Key)) == 1
}

func redirectOrJSON(target *config.Target, w http.ResponseWriter, r *http.Request, code int, message string) {
	if target.Redirect != nil {
		var dest string
		if code >= 400 {
			dest = target.Redirect.Error
		} else {
			dest = target.Redirect.Success
		}
		if dest != "" {
			http.Redirect(w, r, resolveRedirectURL(r, dest), http.StatusSeeOther)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if code >= 400 {
		json.NewEncoder(w).Encode(map[string]string{"error": message})
	} else {
		json.NewEncoder(w).Encode(map[string]string{"status": message})
	}
}

func resolveRedirectURL(r *http.Request, redirectURL string) string {
	parsed, err := url.Parse(redirectURL)
	if err != nil || parsed.IsAbs() {
		return redirectURL
	}
	referer := r.Header.Get("Referer")
	base, err := url.Parse(referer)
	if err != nil || referer == "" {
		return redirectURL
	}
	return base.ResolveReference(parsed).String()
}

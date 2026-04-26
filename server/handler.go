package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/mail"
	"strings"

	"github.com/devopsmitch/go-mail-form/antispam"
	mailpkg "github.com/devopsmitch/go-mail-form/mail"
)

func clientIP(r *http.Request, trustedHeader string) string {
	if trustedHeader != "" {
		if ip := r.Header.Get(trustedHeader); ip != "" {
			// X-Forwarded-For can contain multiple IPs; take the first
			return strings.TrimSpace(strings.Split(ip, ",")[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func sanitizeFilename(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '"' || r == '\\' || r == '\r' || r == '\n' {
			return '_'
		}
		return r
	}, name)
}

func jsonError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (s *Server) mailHandler(w http.ResponseWriter, r *http.Request) {
	targetName := strings.TrimPrefix(r.URL.Path, "/")
	if targetName == "" || targetName == "healthz" || strings.Contains(targetName, "/") {
		jsonError(w, http.StatusNotFound, "not found")
		return
	}

	target, ok := s.Targets[targetName]
	if !ok {
		jsonError(w, http.StatusNotFound, "target not found")
		return
	}

	if cors(target, w, r) {
		return
	}
	if !checkAuth(target, r) {
		redirectOrJSON(target, w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !checkOrigin(target, r) {
		redirectOrJSON(target, w, r, http.StatusForbidden, "origin not allowed")
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ip := clientIP(r, s.TrustedHeader)
	if !s.rateLimitAllow(targetName, ip) {
		jsonError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		if err := r.ParseForm(); err != nil {
			redirectOrJSON(target, w, r, http.StatusBadRequest, "bad request")
			return
		}
	}

	from := r.FormValue("from")
	name := r.FormValue("name")
	subjectPrefix := r.FormValue("subjectPrefix")
	subject := r.FormValue("subject")
	body := r.FormValue("body")

	if antispam.Honeypot(r.FormValue("_gotcha")) {
		redirectOrJSON(target, w, r, http.StatusOK, "ok")
		return
	}

	if verifier, ok := s.Turnstile[targetName]; ok {
		token := r.FormValue("cf-turnstile-response")
		ok, err := verifier.Verify(r.Context(), token, ip)
		if err != nil {
			log.Printf("[!] Turnstile verification error for target %s: %v", targetName, err)
			redirectOrJSON(target, w, r, http.StatusInternalServerError, "captcha verification failed")
			return
		}
		if !ok {
			redirectOrJSON(target, w, r, http.StatusForbidden, "captcha verification failed")
			return
		}
	}

	if problems := validateFields(from, subject, body); len(problems) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"error":    "validation_error",
			"problems": problems,
		}); err != nil {
			log.Printf("[!] Failed to encode validation response: %v", err)
		}
		return
	}

	emailFrom := target.From
	if emailFrom == "" {
		emailFrom = from
	}
	if emailFrom == "" {
		jsonError(w, http.StatusUnprocessableEntity, "from is required when target has no default sender")
		return
	}

	var replyTo string
	if from != "" {
		replyTo = mailpkg.FormatFrom(from, name)
	}

	fullSubject := target.SubjectPrefix + subjectPrefix + subject

	var attachments []mailpkg.Attachment
	if r.MultipartForm != nil {
		for _, fileHeaders := range r.MultipartForm.File {
			for _, fh := range fileHeaders {
				f, err := fh.Open()
				if err != nil {
					jsonError(w, http.StatusBadRequest, "failed to read attachment: "+fh.Filename)
					return
				}
				attachments = append(attachments, mailpkg.Attachment{
					Filename:    sanitizeFilename(fh.Filename),
					ContentType: fh.Header.Get("Content-Type"),
					Data:        f,
				})
			}
		}
	}
	defer func() {
		for _, att := range attachments {
			if c, ok := att.Data.(io.Closer); ok {
				c.Close()
			}
		}
	}()

	if from != "" {
		sender := from
		if name != "" {
			sender = name + " (" + from + ")"
		}
		body = fmt.Sprintf("%s submitted the following via your website:\n\n%s", sender, body)
	}

	if err := s.Sender.SendMail(r.Context(), target, emailFrom, replyTo, fullSubject, body, attachments); err != nil {
		log.Printf("[!] Error sending email for target %s: %v", targetName, err)
		redirectOrJSON(target, w, r, http.StatusInternalServerError, "email sending failed")
		return
	}

	log.Printf("Email sent successfully (target: %s)", targetName)
	redirectOrJSON(target, w, r, http.StatusOK, "ok")
}

func validateFields(from, subject, body string) []map[string]string {
	var problems []map[string]string

	if from != "" {
		if _, err := mail.ParseAddress(from); err != nil {
			problems = append(problems, map[string]string{"field": "from", "error": "invalid email address"})
		}
	}
	subject = strings.TrimSpace(subject)
	if len(subject) < 2 || len(subject) > 255 {
		problems = append(problems, map[string]string{"field": "subject", "error": "required, 2-255 characters"})
	}
	body = strings.TrimSpace(body)
	if len(body) < 5 || len(body) > 32000 {
		problems = append(problems, map[string]string{"field": "body", "error": "required, 5-32000 characters"})
	}
	return problems
}

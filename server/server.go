package server

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/devopsmitch/go-mail-form/config"
	"github.com/devopsmitch/go-mail-form/mail"
)

// MailSender is the interface for sending email.
type MailSender interface {
	SendMail(ctx context.Context, target *config.Target, from, replyTo, subject, body string, attachments []mail.Attachment) error
}

// MailSenderFunc adapts a plain function to the MailSender interface.
type MailSenderFunc func(ctx context.Context, target *config.Target, from, replyTo, subject, body string, attachments []mail.Attachment) error

func (f MailSenderFunc) SendMail(ctx context.Context, target *config.Target, from, replyTo, subject, body string, attachments []mail.Attachment) error {
	return f(ctx, target, from, replyTo, subject, body, attachments)
}

// Server holds all application state.
type Server struct {
	Targets       map[string]*config.Target
	Sender        MailSender
	TrustedHeader string
	limiters      map[string]*targetLimiter
	mu            sync.RWMutex
}

// New creates a Server from loaded targets and starts rate limit cleanup.
func New(targets map[string]*config.Target, sender MailSender) *Server {
	s := &Server{
		Targets:  targets,
		Sender:   sender,
		limiters: make(map[string]*targetLimiter, len(targets)),
	}
	for name, t := range targets {
		s.limiters[name] = &targetLimiter{
			entries: map[string]*rateLimiterEntry{},
			cfg:     t.RateLimit,
		}
	}
	go s.cleanup(5 * time.Minute)
	return s
}

// Handler returns an http.Handler with all routes registered.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthCheck)
	mux.HandleFunc("/", s.mailHandler)
	return mux
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

package server

import (
	"testing"

	"github.com/devopsmitch/go-mail-form/config"
)

func testLimiterServer() *Server {
	targets := map[string]*config.Target{
		"test": {
			SMTP:       "smtps://user:pass@smtp.example.com",
			Recipients: []string{"to@example.com"},
			RateLimit:  &config.RateLimit{Timespan: 60, Requests: 3},
		},
	}
	// Build server without cleanup goroutine for deterministic tests
	s := &Server{
		Targets:  targets,
		limiters: make(map[string]*targetLimiter, len(targets)),
	}
	for name, t := range targets {
		s.limiters[name] = &targetLimiter{
			entries: map[string]*rateLimiterEntry{},
			cfg:     t.RateLimit,
		}
	}
	return s
}

func TestRateLimitAllows(t *testing.T) {
	s := testLimiterServer()
	for i := 0; i < 3; i++ {
		if !s.rateLimitAllow("test", "1.2.3.4") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
}

func TestRateLimitBlocks(t *testing.T) {
	s := testLimiterServer()
	for i := 0; i < 3; i++ {
		s.rateLimitAllow("test", "1.2.3.4")
	}
	if s.rateLimitAllow("test", "1.2.3.4") {
		t.Fatal("4th request should be blocked")
	}
}

func TestRateLimitPerIP(t *testing.T) {
	s := testLimiterServer()
	for i := 0; i < 3; i++ {
		s.rateLimitAllow("test", "1.2.3.4")
	}
	// Different IP should still be allowed
	if !s.rateLimitAllow("test", "5.6.7.8") {
		t.Fatal("different IP should be allowed")
	}
}

func TestRateLimitUnknownTarget(t *testing.T) {
	s := testLimiterServer()
	if s.rateLimitAllow("nonexistent", "1.2.3.4") {
		t.Fatal("unknown target should be denied")
	}
}

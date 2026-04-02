package server

import (
	"sync"
	"time"

	"github.com/devopsmitch/go-mail-form/config"
)

type rateLimiterEntry struct {
	count   int
	resetAt time.Time
}

type targetLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateLimiterEntry
	cfg     *config.RateLimit
}

func (s *Server) rateLimitAllow(targetName, ip string) bool {
	s.mu.RLock()
	tl, ok := s.limiters[targetName]
	s.mu.RUnlock()
	if !ok {
		return false
	}

	tl.mu.Lock()
	defer tl.mu.Unlock()

	now := time.Now()
	e, exists := tl.entries[ip]
	if !exists || now.After(e.resetAt) {
		tl.entries[ip] = &rateLimiterEntry{
			count:   1,
			resetAt: now.Add(time.Duration(tl.cfg.Timespan) * time.Second),
		}
		return true
	}

	if e.count >= tl.cfg.Requests {
		return false
	}

	e.count++
	return true
}

func (s *Server) cleanup(interval time.Duration) {
	for {
		time.Sleep(interval)
		s.mu.RLock()
		for _, tl := range s.limiters {
			tl.mu.Lock()
			now := time.Now()
			for ip, e := range tl.entries {
				if now.After(e.resetAt) {
					delete(tl.entries, ip)
				}
			}
			tl.mu.Unlock()
		}
		s.mu.RUnlock()
	}
}

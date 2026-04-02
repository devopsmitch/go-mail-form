package config

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Target struct {
	SMTP          string     `json:"smtp"`
	Origin        string     `json:"origin"`
	Recipients    []string   `json:"recipients"`
	From          string     `json:"from"`
	SubjectPrefix string     `json:"subjectPrefix"`
	Key           string     `json:"key"`
	Redirect      *Redirect  `json:"redirect"`
	RateLimit     *RateLimit `json:"rateLimit"`
}

type Redirect struct {
	Success string `json:"success"`
	Error   string `json:"error"`
}

type RateLimit struct {
	Timespan int `json:"timespan"`
	Requests int `json:"requests"`
}

// LoadTargets reads all JSON files from dir and returns them keyed by filename (without extension).
func LoadTargets(dir string) (map[string]*Target, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read targets directory %q: %w", dir, err)
	}

	targets := map[string]*Target{}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("cannot read target file %q: %w", e.Name(), err)
		}

		var t Target
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("invalid JSON in %q: %w", e.Name(), err)
		}

		name := strings.TrimSuffix(e.Name(), ".json")

		if err := validateTarget(name, &t); err != nil {
			return nil, err
		}

		targets[name] = &t
		log.Printf("* Loaded target: %s", name)
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no targets found in %q", dir)
	}

	return targets, nil
}

func validateTarget(name string, t *Target) error {
	if t.SMTP == "" {
		return fmt.Errorf("target %q: smtp is required", name)
	}
	u, err := url.Parse(t.SMTP)
	if err != nil || (u.Scheme != "smtp" && u.Scheme != "smtps") {
		return fmt.Errorf("target %q: smtp must be a valid smtp:// or smtps:// URL", name)
	}
	if len(t.Recipients) == 0 {
		return fmt.Errorf("target %q: recipients is required and must not be empty", name)
	}
	if t.RateLimit == nil || t.RateLimit.Timespan <= 0 || t.RateLimit.Requests <= 0 {
		return fmt.Errorf("target %q: rateLimit with positive timespan and requests is required", name)
	}
	return nil
}

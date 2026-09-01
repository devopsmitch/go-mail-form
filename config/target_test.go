package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTargets(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "site.json"), []byte(`{
		"smtp": "smtps://user:pass@smtp.example.com",
		"from": "noreply@example.com",
		"recipients": ["to@example.com"],
		"rateLimit": {"timespan": 60, "requests": 5}
	}`), 0644)

	targets, err := LoadTargets(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	site, ok := targets["site"]
	if !ok {
		t.Fatal("expected target 'site' to be loaded")
	}
	if site.Transport != TransportSMTP {
		t.Errorf("expected transport to default to smtp, got %q", site.Transport)
	}
}

func TestLoadTargetsSES(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "site.json"), []byte(`{
		"transport": "ses",
		"ses": {"region": "ap-southeast-2"},
		"from": "noreply@example.com",
		"recipients": ["to@example.com"],
		"rateLimit": {"timespan": 60, "requests": 5}
	}`), 0644)

	targets, err := LoadTargets(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	site, ok := targets["site"]
	if !ok {
		t.Fatal("expected target 'site' to be loaded")
	}
	if site.Transport != TransportSES {
		t.Errorf("expected ses transport, got %q", site.Transport)
	}
	if site.SES == nil || site.SES.Region != "ap-southeast-2" {
		t.Errorf("expected ses region ap-southeast-2, got %+v", site.SES)
	}
}

func TestLoadTargetsSESMissingRegion(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{
		"transport": "ses",
		"from": "noreply@example.com",
		"recipients": ["to@example.com"],
		"rateLimit": {"timespan": 60, "requests": 5}
	}`), 0644)
	if _, err := LoadTargets(dir); err == nil {
		t.Fatal("expected error for missing ses region")
	}
}

func TestLoadTargetsUnknownTransport(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{
		"transport": "carrierpigeon",
		"from": "noreply@example.com",
		"recipients": ["to@example.com"],
		"rateLimit": {"timespan": 60, "requests": 5}
	}`), 0644)
	if _, err := LoadTargets(dir); err == nil {
		t.Fatal("expected error for unknown transport")
	}
}

func TestLoadTargetsMissingFrom(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{
		"smtp": "smtps://user:pass@smtp.example.com",
		"recipients": ["to@example.com"],
		"rateLimit": {"timespan": 60, "requests": 5}
	}`), 0644)
	if _, err := LoadTargets(dir); err == nil {
		t.Fatal("expected error for missing from")
	}
}

func TestLoadTargetsEmpty(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadTargets(dir)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestLoadTargetsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{not json`), 0644)
	_, err := LoadTargets(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadTargetsMissingSMTP(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{
		"recipients": ["to@example.com"],
		"rateLimit": {"timespan": 60, "requests": 5}
	}`), 0644)
	_, err := LoadTargets(dir)
	if err == nil {
		t.Fatal("expected error for missing smtp")
	}
}

func TestLoadTargetsMissingRecipients(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{
		"smtp": "smtps://user:pass@smtp.example.com",
		"from": "noreply@example.com",
		"rateLimit": {"timespan": 60, "requests": 5}
	}`), 0644)
	_, err := LoadTargets(dir)
	if err == nil {
		t.Fatal("expected error for missing recipients")
	}
}

func TestLoadTargetsMissingRateLimit(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{
		"smtp": "smtps://user:pass@smtp.example.com",
		"from": "noreply@example.com",
		"recipients": ["to@example.com"]
	}`), 0644)
	_, err := LoadTargets(dir)
	if err == nil {
		t.Fatal("expected error for missing rateLimit")
	}
}

func TestLoadTargetsTurnstileEmptySecret(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{
		"smtp": "smtps://user:pass@smtp.example.com",
		"from": "noreply@example.com",
		"recipients": ["to@example.com"],
		"rateLimit": {"timespan": 60, "requests": 5},
		"turnstile": {}
	}`), 0644)
	_, err := LoadTargets(dir)
	if err == nil {
		t.Fatal("expected error for empty turnstile secretKey")
	}
}

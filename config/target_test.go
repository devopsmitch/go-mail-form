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
		"recipients": ["to@example.com"],
		"rateLimit": {"timespan": 60, "requests": 5}
	}`), 0644)

	targets, err := LoadTargets(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := targets["site"]; !ok {
		t.Fatal("expected target 'site' to be loaded")
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
		"recipients": ["to@example.com"],
		"rateLimit": {"timespan": 60, "requests": 5},
		"turnstile": {}
	}`), 0644)
	_, err := LoadTargets(dir)
	if err == nil {
		t.Fatal("expected error for empty turnstile secretKey")
	}
}

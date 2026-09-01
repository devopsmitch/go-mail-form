package mail

import (
	"strings"
	"testing"
)

func TestFormatFrom(t *testing.T) {
	tests := []struct {
		email, name, want string
	}{
		{"user@example.com", "", "user@example.com"},
		{"user@example.com", "John Doe", "John Doe <user@example.com>"},
		{"user@example.com", "John", "John <user@example.com>"},
		{"user@example.com", "  ", "user@example.com"},
		// Non-ASCII display name is RFC 2047 encoded.
		{"user@example.com", "Café", "=?utf-8?q?Caf=C3=A9?= <user@example.com>"},
		// CRLF in an otherwise-ASCII name is stripped (no encoding needed).
		{"user@example.com", "Bad\r\nName", "BadName <user@example.com>"},
	}
	for _, tt := range tests {
		got := FormatFrom(tt.email, tt.name)
		if got != tt.want {
			t.Errorf("FormatFrom(%q, %q) = %q, want %q", tt.email, tt.name, got, tt.want)
		}
	}
}

func TestSanitizeHeader(t *testing.T) {
	if got := sanitizeHeader("a\r\nb"); got != "ab" {
		t.Errorf("sanitizeHeader stripped CRLF incorrectly: got %q", got)
	}
	if got := sanitizeHeader("plain"); got != "plain" {
		t.Errorf("sanitizeHeader altered plain text: got %q", got)
	}
}

func TestQuoteFilename(t *testing.T) {
	tests := []struct{ in, want string }{
		{"report.pdf", "report.pdf"},
		{"a\r\nb.txt", "ab.txt"},
		{`evil".txt`, "evil.txt"},
		{`back\slash.txt`, "backslash.txt"},
		{`"; name="x`, "; name=x"},
	}
	for _, tt := range tests {
		if got := quoteFilename(tt.in); got != tt.want {
			t.Errorf("quoteFilename(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBuildMessageHeaderInjection(t *testing.T) {
	// A subject attempting to inject a Bcc header must not produce an extra header line.
	msg, err := buildMessage(
		"sender@example.com",
		"",
		[]string{"to@example.com"},
		"Hello\r\nBcc: victim@example.com",
		"body text here",
		nil,
	)
	if err != nil {
		t.Fatalf("buildMessage returned error: %v", err)
	}
	// The injected content must not appear as its own header line (i.e. right
	// after a CRLF). Folded into an existing line it is harmless.
	if strings.Contains(msg, "\r\nBcc:") {
		t.Errorf("header injection not prevented; message contains injected Bcc header line:\n%s", msg)
	}
	// The whole subject value should collapse onto a single Subject line.
	if !strings.Contains(msg, "Subject: HelloBcc: victim@example.com\r\n") {
		t.Errorf("subject CRLF was not stripped as expected:\n%s", msg)
	}
}

func TestBuildMessageNonASCIISubject(t *testing.T) {
	msg, err := buildMessage(
		"sender@example.com",
		"",
		[]string{"to@example.com"},
		"Café ☕",
		"body text here",
		nil,
	)
	if err != nil {
		t.Fatalf("buildMessage returned error: %v", err)
	}
	if !strings.Contains(msg, "Subject: =?utf-8?") {
		t.Errorf("non-ASCII subject was not RFC 2047 encoded:\n%s", msg)
	}
}

func TestBuildMessageAttachmentHeaderInjection(t *testing.T) {
	att := Attachment{
		Filename:    "file\r\nX-Injected: yes.txt",
		ContentType: "text/plain\r\nX-Evil: 1",
		Data:        strings.NewReader("data"),
	}
	msg, err := buildMessage(
		"sender@example.com",
		"",
		[]string{"to@example.com"},
		"subject",
		"body text here",
		[]Attachment{att},
	)
	if err != nil {
		t.Fatalf("buildMessage returned error: %v", err)
	}
	// Injected content must not begin a new header line.
	if strings.Contains(msg, "\r\nX-Injected:") || strings.Contains(msg, "\r\nX-Evil:") {
		t.Errorf("attachment header injection not prevented:\n%s", msg)
	}
}

func TestBuildMessageAttachmentFilenameQuoteBreakout(t *testing.T) {
	// A filename that tries to close the quoted string and inject a new
	// parameter must not succeed.
	att := Attachment{
		Filename:    `x"; name="evil`,
		ContentType: "text/plain",
		Data:        strings.NewReader("data"),
	}
	msg, err := buildMessage(
		"sender@example.com",
		"",
		[]string{"to@example.com"},
		"subject",
		"body text here",
		[]Attachment{att},
	)
	if err != nil {
		t.Fatalf("buildMessage returned error: %v", err)
	}
	if strings.Contains(msg, `name="evil`) {
		t.Errorf("filename quote breakout not prevented:\n%s", msg)
	}
	// The sanitized filename should remain within a single quoted value.
	if !strings.Contains(msg, `filename="x; name=evil"`) {
		t.Errorf("unexpected filename encoding:\n%s", msg)
	}
}

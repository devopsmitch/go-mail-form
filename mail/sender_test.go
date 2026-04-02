package mail

import "testing"

func TestFormatFrom(t *testing.T) {
	tests := []struct {
		email, name, want string
	}{
		{"user@example.com", "", "user@example.com"},
		{"user@example.com", "John Doe", "John Doe <user@example.com>"},
		{"user@example.com", "John", "John <user@example.com>"},
		{"user@example.com", "  ", "user@example.com"},
	}
	for _, tt := range tests {
		got := FormatFrom(tt.email, tt.name)
		if got != tt.want {
			t.Errorf("FormatFrom(%q, %q) = %q, want %q", tt.email, tt.name, got, tt.want)
		}
	}
}

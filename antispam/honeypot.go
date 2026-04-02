package antispam

// Honeypot checks if a honeypot field was filled in (indicating a bot).
// Pass the value of the hidden honeypot form field.
func Honeypot(value string) bool {
	return value != ""
}

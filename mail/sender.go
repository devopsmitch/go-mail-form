package mail

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net"
	"net/smtp"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/devopsmitch/go-mail-form/config"
)

const sendTimeout = 30 * time.Second

// Attachment represents an uploaded file to attach to the email.
type Attachment struct {
	Filename    string
	ContentType string
	Data        io.Reader
}

// FormatFrom builds a "Name <email>" string from the parts provided.
func FormatFrom(email, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return email
	}
	return fmt.Sprintf("%s <%s>", name, email)
}

// SendMail sends an email for the given target.
func SendMail(ctx context.Context, target *config.Target, from, replyTo, subject, body string, attachments []Attachment) error {
	u, err := url.Parse(target.SMTP)
	if err != nil {
		return fmt.Errorf("invalid smtp url: %w", err)
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		if u.Scheme == "smtps" {
			port = "465"
		} else {
			port = "587"
		}
	}
	addr := net.JoinHostPort(host, port)

	password, _ := u.User.Password()
	username := u.User.Username()

	to := target.Recipients

	msg, err := buildMessage(from, replyTo, to, subject, body, attachments)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	dialer := &net.Dialer{}
	var conn net.Conn
	if u.Scheme == "smtps" {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: host})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	// Set a deadline on the connection so SMTP commands don't hang
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	if u.Scheme != "smtps" {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
				return fmt.Errorf("starttls: %w", err)
			}
		}
	}

	if username != "" {
		auth := smtp.PlainAuth("", username, password, host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}

	if err := c.Mail(from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	for _, r := range to {
		if err := c.Rcpt(r); err != nil {
			return fmt.Errorf("rcpt to: %w", err)
		}
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}

	return c.Quit()
}

func generateBoundary() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("----=_GoMailForm_%x", b)
}

func buildMessage(from, replyTo string, to []string, subject, body string, attachments []Attachment) (string, error) {
	var b strings.Builder

	b.WriteString("From: " + from + "\r\n")
	if replyTo != "" {
		b.WriteString("Reply-To: " + replyTo + "\r\n")
	}
	b.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")

	if len(attachments) == 0 {
		b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		b.WriteString("\r\n")
		b.WriteString(body)
		return b.String(), nil
	}

	boundary := generateBoundary()
	b.WriteString("Content-Type: multipart/mixed; boundary=\"" + boundary + "\"\r\n")
	b.WriteString("\r\n")

	// Body part
	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(body + "\r\n")

	// Attachments
	for _, att := range attachments {
		data, err := io.ReadAll(att.Data)
		if err != nil {
			return "", fmt.Errorf("read attachment %q: %w", att.Filename, err)
		}
		ct := att.ContentType
		if ct == "" {
			ct = mime.TypeByExtension(filepath.Ext(att.Filename))
			if ct == "" {
				ct = "application/octet-stream"
			}
		}
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString("Content-Type: " + ct + "\r\n")
		b.WriteString("Content-Disposition: attachment; filename=\"" + att.Filename + "\"\r\n")
		b.WriteString("Content-Transfer-Encoding: base64\r\n")
		b.WriteString("\r\n")
		b.WriteString(base64.StdEncoding.EncodeToString(data) + "\r\n")
	}

	b.WriteString("--" + boundary + "--\r\n")
	return b.String(), nil
}

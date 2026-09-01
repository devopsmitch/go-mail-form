package server

import (
	"context"
	"testing"

	"github.com/devopsmitch/go-mail-form/config"
	"github.com/devopsmitch/go-mail-form/mail"
)

func TestDispatcherRoutesByTransport(t *testing.T) {
	var smtpCalled, sesCalled bool
	smtp := MailSenderFunc(func(ctx context.Context, target *config.Target, from, replyTo, subject, body string, attachments []mail.Attachment) error {
		smtpCalled = true
		return nil
	})
	ses := MailSenderFunc(func(ctx context.Context, target *config.Target, from, replyTo, subject, body string, attachments []mail.Attachment) error {
		sesCalled = true
		return nil
	})

	d := NewDispatcher(map[config.Transport]MailSender{
		config.TransportSMTP: smtp,
		config.TransportSES:  ses,
	})

	smtpTarget := &config.Target{Transport: config.TransportSMTP}
	if err := d.SendMail(context.Background(), smtpTarget, "f", "", "s", "b", nil); err != nil {
		t.Fatalf("smtp dispatch error: %v", err)
	}
	if !smtpCalled || sesCalled {
		t.Errorf("expected only smtp sender to be called; smtp=%v ses=%v", smtpCalled, sesCalled)
	}

	smtpCalled, sesCalled = false, false
	sesTarget := &config.Target{Transport: config.TransportSES}
	if err := d.SendMail(context.Background(), sesTarget, "f", "", "s", "b", nil); err != nil {
		t.Fatalf("ses dispatch error: %v", err)
	}
	if !sesCalled || smtpCalled {
		t.Errorf("expected only ses sender to be called; smtp=%v ses=%v", smtpCalled, sesCalled)
	}
}

func TestDispatcherUnknownTransport(t *testing.T) {
	d := NewDispatcher(map[config.Transport]MailSender{})
	target := &config.Target{Transport: config.TransportSMTP}
	if err := d.SendMail(context.Background(), target, "f", "", "s", "b", nil); err == nil {
		t.Fatal("expected error when no sender configured for transport")
	}
}

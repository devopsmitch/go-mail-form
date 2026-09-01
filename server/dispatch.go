package server

import (
	"context"
	"fmt"

	"github.com/devopsmitch/go-mail-form/config"
	"github.com/devopsmitch/go-mail-form/mail"
)

// Dispatcher routes each SendMail call to the sender configured for the
// target's transport. It implements MailSender so it can be used as the
// server's single Sender while supporting multiple transports.
type Dispatcher struct {
	byTransport map[config.Transport]MailSender
}

// NewDispatcher builds a Dispatcher from a map of transport to sender.
func NewDispatcher(senders map[config.Transport]MailSender) *Dispatcher {
	return &Dispatcher{byTransport: senders}
}

// SendMail selects the sender for the target's transport and delegates to it.
func (d *Dispatcher) SendMail(ctx context.Context, target *config.Target, from, replyTo, subject, body string, attachments []mail.Attachment) error {
	sender, ok := d.byTransport[target.Transport]
	if !ok {
		return fmt.Errorf("no sender configured for transport %q", target.Transport)
	}
	return sender.SendMail(ctx, target, from, replyTo, subject, body, attachments)
}

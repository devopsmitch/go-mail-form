package mail

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"

	"github.com/devopsmitch/go-mail-form/config"
)

// sesAPI is the subset of the SES v2 client used by SESSender.
// It is defined as an interface so tests can supply a fake.
type sesAPI interface {
	SendEmail(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

// SESSender sends mail through Amazon SES using the SendEmail API with a raw
// MIME payload. It reuses buildMessage so the same message construction and
// header sanitization apply to both SMTP and SES transports.
type SESSender struct {
	client sesAPI
}

// NewSESSender builds an SESSender for the given AWS region using the ambient
// AWS credential chain (environment, shared config, IAM role).
func NewSESSender(ctx context.Context, region string) (*SESSender, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	return &SESSender{client: sesv2.NewFromConfig(cfg)}, nil
}

// SendMail implements the server.MailSender interface. It sends a raw MIME
// message built by buildMessage. FromEmailAddress is set explicitly from the
// from argument; recipients are taken from the message's To header (written by
// buildMessage), so Destination is intentionally left unset.
func (s *SESSender) SendMail(ctx context.Context, target *config.Target, from, replyTo, subject, body string, attachments []Attachment) error {
	msg, err := buildMessage(from, replyTo, target.Recipients, subject, body, attachments)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	_, err = s.client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: &from,
		Content: &types.EmailContent{
			Raw: &types.RawMessage{Data: []byte(msg)},
		},
	})
	if err != nil {
		return fmt.Errorf("ses send: %w", err)
	}
	return nil
}

// SESRouter dispatches to a per-region SESSender based on the target's
// configured region. This lets a single deployment serve SES targets across
// multiple AWS regions.
type SESRouter struct {
	byRegion map[string]*SESSender
}

// NewSESRouter creates an empty router.
func NewSESRouter() *SESRouter {
	return &SESRouter{byRegion: map[string]*SESSender{}}
}

// Add registers a sender for a region.
func (r *SESRouter) Add(region string, s *SESSender) {
	r.byRegion[region] = s
}

// SendMail routes to the sender for the target's region.
func (r *SESRouter) SendMail(ctx context.Context, target *config.Target, from, replyTo, subject, body string, attachments []Attachment) error {
	if target.SES == nil {
		return fmt.Errorf("ses transport selected but target has no ses config")
	}
	s, ok := r.byRegion[target.SES.Region]
	if !ok {
		return fmt.Errorf("no ses sender for region %q", target.SES.Region)
	}
	return s.SendMail(ctx, target, from, replyTo, subject, body, attachments)
}

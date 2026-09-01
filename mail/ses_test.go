package mail

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sesv2"

	"github.com/devopsmitch/go-mail-form/config"
)

type fakeSES struct {
	input *sesv2.SendEmailInput
	err   error
}

func (f *fakeSES) SendEmail(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	f.input = params
	if f.err != nil {
		return nil, f.err
	}
	return &sesv2.SendEmailOutput{}, nil
}

func TestSESSenderSendMail(t *testing.T) {
	fake := &fakeSES{}
	s := &SESSender{client: fake}
	target := &config.Target{
		Transport:  config.TransportSES,
		SES:        &config.SES{Region: "ap-southeast-2"},
		From:       "noreply@example.com",
		Recipients: []string{"to@example.com"},
	}

	err := s.SendMail(context.Background(), target, target.From, "visitor@example.com", "Hello", "body text here", nil)
	if err != nil {
		t.Fatalf("SendMail returned error: %v", err)
	}
	if fake.input == nil || fake.input.Content == nil || fake.input.Content.Raw == nil {
		t.Fatal("expected a raw email content to be sent")
	}
	raw := string(fake.input.Content.Raw.Data)
	if !strings.Contains(raw, "From: noreply@example.com") {
		t.Errorf("raw message missing From header:\n%s", raw)
	}
	if !strings.Contains(raw, "To: to@example.com") {
		t.Errorf("raw message missing To header:\n%s", raw)
	}
	if !strings.Contains(raw, "Subject: Hello") {
		t.Errorf("raw message missing Subject header:\n%s", raw)
	}
	if !strings.Contains(raw, "Reply-To: visitor@example.com") {
		t.Errorf("raw message missing Reply-To header:\n%s", raw)
	}
	if fake.input.FromEmailAddress == nil || *fake.input.FromEmailAddress != "noreply@example.com" {
		t.Errorf("expected FromEmailAddress to be set to the sender, got %v", fake.input.FromEmailAddress)
	}
}

func TestSESSenderSendMailError(t *testing.T) {
	fake := &fakeSES{err: errors.New("boom")}
	s := &SESSender{client: fake}
	target := &config.Target{
		SES:        &config.SES{Region: "us-east-1"},
		From:       "noreply@example.com",
		Recipients: []string{"to@example.com"},
	}
	err := s.SendMail(context.Background(), target, target.From, "", "subject", "body text here", nil)
	if err == nil {
		t.Fatal("expected error to propagate from SES client")
	}
}

func TestSESRouter(t *testing.T) {
	fake := &fakeSES{}
	router := NewSESRouter()
	router.Add("us-east-1", &SESSender{client: fake})

	target := &config.Target{
		SES:        &config.SES{Region: "us-east-1"},
		From:       "noreply@example.com",
		Recipients: []string{"to@example.com"},
	}
	if err := router.SendMail(context.Background(), target, target.From, "", "subject", "body text here", nil); err != nil {
		t.Fatalf("router SendMail returned error: %v", err)
	}
	if fake.input == nil {
		t.Fatal("expected router to delegate to region sender")
	}

	// Unknown region should error.
	target.SES.Region = "eu-west-1"
	if err := router.SendMail(context.Background(), target, target.From, "", "subject", "body text here", nil); err == nil {
		t.Fatal("expected error for unregistered region")
	}
}

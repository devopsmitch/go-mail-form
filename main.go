package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/devopsmitch/go-mail-form/config"
	"github.com/devopsmitch/go-mail-form/mail"
	"github.com/devopsmitch/go-mail-form/server"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

// envOr returns the value of the environment variable key, or def if unset/empty.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// buildSender constructs a transport dispatcher for the loaded targets. SMTP is
// always available; SES clients are created per distinct region, and only when
// at least one target uses the SES transport (so SMTP-only deployments need no
// AWS credentials).
func buildSender(ctx context.Context, targets map[string]*config.Target) (server.MailSender, error) {
	senders := map[config.Transport]server.MailSender{
		config.TransportSMTP: server.MailSenderFunc(mail.SendMail),
	}

	// Collect distinct SES regions in use.
	regions := map[string]struct{}{}
	for _, t := range targets {
		if t.Transport == config.TransportSES && t.SES != nil {
			regions[t.SES.Region] = struct{}{}
		}
	}

	if len(regions) > 0 {
		router := mail.NewSESRouter()
		for region := range regions {
			s, err := mail.NewSESSender(ctx, region)
			if err != nil {
				return nil, fmt.Errorf("ses region %q: %w", region, err)
			}
			router.Add(region, s)
			log.Printf("* SES transport ready for region %s", region)
		}
		senders[config.TransportSES] = router
	}

	return server.NewDispatcher(senders), nil
}

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	port := flag.String("port", envOr("PORT", "3000"), "port to listen on (overrides PORT)")
	targetsDir := flag.String("targets-dir", envOr("TARGETS_DIR", "targets"), "directory containing target JSON files (overrides TARGETS_DIR)")
	trustedHeader := flag.String("trusted-header", envOr("TRUSTED_HEADER", ""), "HTTP header to trust for client IP, e.g. CF-Connecting-IP (overrides TRUSTED_HEADER)")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}

	targets, err := config.LoadTargets(*targetsDir)
	if err != nil {
		log.Fatalf("Failed to load targets: %v", err)
	}

	sender, err := buildSender(context.Background(), targets)
	if err != nil {
		log.Fatalf("Failed to initialize mail senders: %v", err)
	}
	srv := server.New(targets, sender)
	srv.TrustedHeader = *trustedHeader

	addr := ":" + *port
	log.Printf("MailForm %s started on port %s", version, *port)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}

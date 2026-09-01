package main

import (
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

	sender := server.MailSenderFunc(mail.SendMail)
	srv := server.New(targets, sender)
	srv.TrustedHeader = *trustedHeader

	addr := ":" + *port
	log.Printf("MailForm %s started on port %s", version, *port)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}

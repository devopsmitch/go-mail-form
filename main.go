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

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	targetsDir := os.Getenv("TARGETS_DIR")
	if targetsDir == "" {
		targetsDir = "targets"
	}

	targets, err := config.LoadTargets(targetsDir)
	if err != nil {
		log.Fatalf("Failed to load targets: %v", err)
	}

	sender := server.MailSenderFunc(mail.SendMail)
	srv := server.New(targets, sender)
	srv.TrustedHeader = os.Getenv("TRUSTED_HEADER")

	addr := ":" + port
	log.Printf("MailForm %s started on port %s", version, port)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}

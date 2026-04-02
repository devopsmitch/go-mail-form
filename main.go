package main

import (
	"log"
	"net/http"
	"os"

	"github.com/devopsmitch/go-mail-form/config"
	"github.com/devopsmitch/go-mail-form/mail"
	"github.com/devopsmitch/go-mail-form/server"
)

func main() {
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
	log.Printf("MailForm started on port %s", port)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}

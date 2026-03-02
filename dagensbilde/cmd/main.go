package main

import (
	"log"
	"os"
	"syscall"

	"github.com/jesperkha/dagensbilde/config"
	"github.com/jesperkha/dagensbilde/database"
	"github.com/jesperkha/dagensbilde/server"
	"github.com/jesperkha/notifier"
)

func main() {
	cfg := config.Load()
	notif := notifier.New()

	db, err := database.New(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Migrate("sql"); err != nil {
		log.Fatal(err)
	}

	srv := server.New(cfg, db)
	go srv.ListenAndServe(notif)

	notif.NotifyOnSignal(os.Interrupt, syscall.SIGTERM)
	log.Println("shutdown complete")
}

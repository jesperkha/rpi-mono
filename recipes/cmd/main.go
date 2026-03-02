package main

import (
	"log"
	"os"
	"syscall"

	"github.com/jesperkha/recipes/config"
	"github.com/jesperkha/recipes/server"
	"github.com/jesperkha/notifier"
)

func main() {
	config := config.Load()
	notif := notifier.New()

	server := server.New(config)
	go server.ListenAndServe(notif)

	notif.NotifyOnSignal(os.Interrupt, syscall.SIGTERM)
	log.Println("shutdown complete")
}

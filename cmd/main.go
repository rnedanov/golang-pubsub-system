package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rnedanov/golang-pubsub-system/internal/config"
	"github.com/rnedanov/golang-pubsub-system/internal/server"
)

func main() {
	// Init config
	cfg, err := config.InitServiceConfig()
	if err != nil {
		log.Fatalf("Error initializing service config: %v", err)
	}

	// Create and start server
	srv := server.NewServer(cfg)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	srv.Stop()
}

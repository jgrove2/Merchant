package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"backend/internal/db"
)

func main() {
	log.Println("Starting Trader Service...")

	// 1. Initialize DB (to log trade executions)
	_, err := db.Connect()
	if err != nil {
		log.Fatalf("Could not connect to DB: %v", err)
	}

	// 2. Setup Context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down Trader...")
		cancel()
	}()

	// 3. Listen for Trade Signals
	log.Println("Waiting for trade signals from Pub/Sub...")

	// TODO: Initialize Redis/NATS Subscriber here

	// Keep the service alive
	<-ctx.Done()
	log.Println("Trader service gracefully stopped.")
}

func executeTrade(opportunityID uint) {
	// TODO: Implement the algorithm to commit the trade to Kalshi
}

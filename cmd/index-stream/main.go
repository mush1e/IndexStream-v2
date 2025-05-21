package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mush1e/IndexStream-v2/config"
	"github.com/mush1e/IndexStream-v2/internal/server"
	"github.com/mush1e/IndexStream-v2/internal/service"
)

func main() {
	cfg := config.Get()
	srv := server.NewServer(cfg)

	go service.ExtractText()

	go func() {
		log.Printf("Server starting on %s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Printf("Shutdown signal received (%v), initiating graceful shutdown...\n", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v\n", err)
	}
	close(service.IndexTargetChan)
	log.Println("Server gracefully stopped")
}

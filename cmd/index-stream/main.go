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
	log.Println("🔍 Starting IndexStream-v2...")

	cfg := config.Get()
	srv := server.NewServer(cfg)

	// Start the text extraction service
	go service.ExtractText()

	// Prewarm cache on startup (after a brief delay)
	go func() {
		time.Sleep(2 * time.Second)
		log.Println("Prewarming cache with frequent terms...")
		service.InvertedIndex.PrewarmCache()
	}()

	// Start the HTTP server
	go func() {
		log.Printf("🚀 Server starting on %s", srv.Addr)
		log.Printf("📖 Open http://localhost%s in your browser", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Printf("🛑 Shutdown signal received (%v), initiating graceful shutdown...", sig)

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	log.Println("🔄 Shutting down HTTP server...")
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("⚠️  Server forced to shutdown: %v", err)
	}

	// Shutdown text extractor
	log.Println("🔄 Shutting down text extractor...")
	service.ShutdownExtractor()

	// Print final statistics
	stats := service.InvertedIndex.GetIndexStats()
	log.Printf("📊 Final Statistics:")
	log.Printf("   Documents indexed: %v", stats["total_documents"])
	log.Printf("   Unique terms: %v", stats["unique_terms"])
	log.Printf("   Average document length: %.2f", stats["average_doc_length"])

	cacheStats := service.InvertedIndex.GetCacheStats()
	if cacheInfo, ok := cacheStats["stats"]; ok {
		log.Printf("💾 Cache Statistics:")
		log.Printf("   L1 Cache: %v", cacheInfo)
	}

	log.Println("✅ Server gracefully stopped")
}

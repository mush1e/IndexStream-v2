package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/mush1e/IndexStream-v2/config"
	"github.com/mush1e/IndexStream-v2/internal/handler"
	"github.com/mush1e/IndexStream-v2/internal/middleware"
)

func NewServer(cfg *config.Config) *http.Server {
	mux := http.NewServeMux()
	addr := ":" + strconv.Itoa(cfg.Port)

	// Main routes
	mux.HandleFunc("/", handler.GetHome)
	mux.HandleFunc("GET /search", handler.GetSearch)
	mux.HandleFunc("GET /crawl", handler.GetCrawl)
	mux.HandleFunc("POST /crawl", handler.PostCrawl)

	// Statistics and monitoring
	mux.HandleFunc("GET /stats", handler.GetStats)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
	})

	// Cache management endpoints
	mux.HandleFunc("POST /cache/clear", handler.PostClearCache)
	mux.HandleFunc("POST /cache/prewarm", handler.PostPrewarmCache)
	mux.HandleFunc("POST /cache/optimize", handler.PostOptimizeCache)

	// Add CORS middleware for API endpoints
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Apply middleware
	loggedMux := middleware.Logger(corsMiddleware(mux))

	return &http.Server{
		Addr:         addr,
		Handler:      loggedMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

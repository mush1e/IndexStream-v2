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

	mux.HandleFunc("/", handler.GetHome)
	mux.HandleFunc("GET /search", handler.GetSearch)
	mux.HandleFunc("GET /crawl", handler.GetCrawl)
	mux.HandleFunc("POST /crawl", handler.PostCrawl)

	loggedMux := middleware.Logger(mux)

	return &http.Server{
		Addr:         addr,
		Handler:      loggedMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

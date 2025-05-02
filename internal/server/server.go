package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/mush1e/IndexStream-v2/config"
)

func NewServer(cfg *config.Config) *http.Server {

	mux := http.NewServeMux()
	addr := ":" + strconv.Itoa(cfg.Port)

	return &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

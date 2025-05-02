package main

import (
	"log"

	"github.com/mush1e/IndexStream-v2/config"
	"github.com/mush1e/IndexStream-v2/internal/server"
)

func main() {
	cfg := config.Load()
	srv := server.NewServer(cfg)

	log.Printf("Listening on %v\n", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}

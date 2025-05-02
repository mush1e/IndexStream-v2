package handler

import (
	"fmt"
	"net/http"
)

func GetHome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "🪿 GooseSearch is live!\n")
}

func GetSearch(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Search endpoint coming soon!\n")
}

func GetCrawl(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Crawl endpoint coming soon!")
}

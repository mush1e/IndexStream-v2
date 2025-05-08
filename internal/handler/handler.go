package handler

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/mush1e/IndexStream-v2/internal/service"
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

func PostCrawl(w http.ResponseWriter, r *http.Request) {
	crawl_url := r.URL.Query().Get("url")

	if crawl_url == "" {
		http.Error(w, "invalid query: missing 'url' parameter", http.StatusBadRequest)
		return
	}

	if u, err := url.ParseRequestURI(crawl_url); err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		http.Error(w, "bad URL provided", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("crawl has been queued for " + crawl_url))

	go func(crawl_url string) {
		service.Crawl(crawl_url)
	}(crawl_url)
}

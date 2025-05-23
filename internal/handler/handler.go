package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/mush1e/IndexStream-v2/internal/service"
)

func GetHome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "🪿 GooseSearch is live!\n")
}

func GetSearch(w http.ResponseWriter, r *http.Request) {
	searchQuery := r.URL.Query().Get("search-query")
	if searchQuery == "" {
		http.Error(w, "invalid query missing 'search-query' parameter", http.StatusBadRequest)
	}

	searchLimit, err := strconv.Atoi(r.URL.Query().Get("k"))
	if err != nil || searchLimit < 0 {
		searchLimit = 10
	}

	searchResults := service.InvertedIndex.Search(searchQuery, searchLimit)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	if err := enc.Encode(searchResults); err != nil {
		http.Error(w, "failed to json encode search results", http.StatusInternalServerError)
	}
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
		service.CrawlRecursive(crawl_url)
	}(crawl_url)
}

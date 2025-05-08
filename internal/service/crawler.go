package service

import (
	"io"
	"log"
	"net/http"
)

func Crawl(url string) {
	log.Printf("Starting crawl on %q\n", url)
	defer log.Printf("Finished crawling %q\n", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to fetch %q\n\tnerr : %v\n", url, err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Non-OK status for %q : %v", url, resp.StatusCode)
		return
	}

	body_content, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Error reading content of &q\n\terr : %v\n", url, err)
	}

	defer resp.Body.Close()

	log.Printf("%v\n", string(body_content))
}

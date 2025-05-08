package service

import "log"

func Crawl(url string) {
	log.Printf("Starting crawl on %q\n", url)
	// Do stuff
	log.Printf("Finished crawling %q\n", url)
}

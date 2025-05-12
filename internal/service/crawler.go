package service

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/mush1e/IndexStream-v2/config"
)

var cfg config.Config = *config.Get()

func hashAndStore(url string, file_contents []byte) error {
	sum := sha256.Sum256([]byte(url))
	docID := hex.EncodeToString(sum[:])

	if err := os.MkdirAll(cfg.DataURL, 0755); err != nil {
		log.Printf("Error generating data dump dir\n\terr : %v\n", err)
		return err
	}

	file_path := filepath.Join(cfg.DataURL, docID+".html")
	if err := os.WriteFile(file_path, file_contents, 0644); err != nil {
		log.Printf("Error writing to dump file\n\terr : %v\n", err)
		return err
	}
	log.Printf("Saved %s.html for %q", docID, url)
	return nil
}

func Crawl(url string) (map[string]struct{}, error) {

	log.Printf("Starting crawl on %q\n", url)
	defer log.Printf("Finished crawling %q\n", url)

	resp, err := http.Get(url)

	if err != nil {
		log.Printf("Failed to fetch %q\n\tnerr : %v\n", url, err)
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Non-OK status for %q : %v", url, resp.StatusCode)
		return nil, err
	}

	body_content, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Error reading content of %q\n\terr : %v\n", url, err)
		return nil, err
	}

	if err := hashAndStore(url, body_content); err != nil {
		return nil, err
	}

	urlList, err := URLExtractor(body_content, url)
	if err != nil {
		return nil, err
	}

	return urlList, nil
}

// This function has borderline excessive documentation since its a bit complicated
func CrawlRecursive(seedURL string) {

	// Retrieve MAX CRAWL DEPTH from config struct
	maxDepth := cfg.SearchDepth

	// A hash set of visited urls to dedupe results
	visitedURLs := make(map[string]struct{})

	// mutex to safely have multiple go routines update visitedURLS
	var mu sync.RWMutex

	// Created a wait group to wait for all go routines to end before returning from function
	var wg sync.WaitGroup

	// A semaphore of size 10 to limit the number of concurrent go routines to 10
	sem := make(chan struct{}, 10)

	// declare function before defining it so that recursive calls inside dont cause errors due to function not being bound to arguments and stuff
	var worker func(string, int)

	// recursive helper function to actually recursively crawl to depth
	worker = func(url string, curr_depth int) {

		// Base case to terminate recursion
		if curr_depth > maxDepth {
			return
		}

		// locking reads and writes on the visitedURLs hash set
		mu.Lock()

		// if the current url passed to worker already exists in our hash set unlock mutex and return
		if _, seen := visitedURLs[url]; seen {
			mu.Unlock()
			return
		}

		// Otherwise add the new url to our hash set and unlock our mutex
		visitedURLs[url] = struct{}{}
		mu.Unlock()

		// signal to the WaitGroup that one new goroutine needs to be monitored and start go routine
		wg.Add(1)
		go func() {

			// Once goroutine concludes we gotta tell the WaitGroup that one of the go routines have finished and to reduce count by one
			defer wg.Done()

			// Add one <insert right work> to the semaphore channel signifying another go routine has started
			// If theres already 10 entries in our semaphone channel this will block the next go routine till a go routine finishes working and we have 9 in the channel buffer
			sem <- struct{}{}

			// To release one entry from our channel buffer
			defer func() { <-sem }()

			// The meat and potatoes of our code, now we want to start crawling our url and get all valid vistable urls from here
			extracted_urls, err := Crawl(url)

			// if we have some trouble in this process get outta this go routine
			if err != nil {
				return
			}

			// We iterate over all our extracted urls and call worker recursively with our depth increasing by one
			for u := range extracted_urls {
				worker(u, curr_depth+1)
			}
		}()
	}

	// init worker with seed URL
	worker(seedURL, 0)

	// wait till all our go routines are done working
	wg.Wait()
}

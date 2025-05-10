package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mush1e/IndexStream-v2/config"
)

type URL struct {
}

func hashAndStore(url string, file_contents []byte) error {
	cfg := config.Get()
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

func Crawl(url string) error {

	log.Printf("Starting crawl on %q\n", url)
	defer log.Printf("Finished crawling %q\n", url)

	resp, err := http.Get(url)

	if err != nil {
		log.Printf("Failed to fetch %q\n\tnerr : %v\n", url, err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Non-OK status for %q : %v", url, resp.StatusCode)
		return err
	}

	body_content, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Printf("Error reading content of %q\n\terr : %v\n", url, err)
		return err
	}

	if err := hashAndStore(url, body_content); err != nil {
		return err
	}

	urlList, err := URLExtractor(body_content, url)
	if err != nil {
		return err
	}

	for k := range urlList {
		fmt.Printf("collected url - %q\n", k)
	}

	return nil
}

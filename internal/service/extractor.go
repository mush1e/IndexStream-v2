package service

import (
	"bytes"
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

var IndexTargetChan = make(chan string, 100) // Increased buffer size
var extractorCtx context.Context
var extractorCancel context.CancelFunc
var extractorWg sync.WaitGroup

func init() {
	extractorCtx, extractorCancel = context.WithCancel(context.Background())
}

func parseHTML(htmlBytes *[]byte) string {
	var rawText strings.Builder
	htmlReader := bytes.NewReader(*htmlBytes)
	node, err := html.Parse(htmlReader)
	if err != nil {
		log.Printf("error parsing html : %q\n", err)
		return ""
	}

	visit := func(node *html.Node) {
		if node.Type == html.TextNode {
			// Clean and normalize text
			text := strings.TrimSpace(node.Data)
			if text != "" {
				rawText.WriteString(text)
				rawText.WriteString(" ")
			}
		}
	}
	traverseDOMTree(node, visit)
	return strings.TrimSpace(rawText.String())
}

func ExtractText() {
	sem := make(chan struct{}, 5) // Semaphore for limiting concurrent processing
	defer close(sem)

	log.Println("Text extractor started")

	for {
		select {
		case filePath := <-IndexTargetChan:
			// Acquire semaphore
			sem <- struct{}{}
			extractorWg.Add(1)

			go func(filePath string) {
				defer extractorWg.Done()
				defer func() { <-sem }()

				processFile(filePath)
			}(filePath)

		case <-extractorCtx.Done():
			log.Println("Text extractor shutting down...")
			extractorWg.Wait() // Wait for all ongoing processing to complete
			log.Println("Text extractor stopped")
			return
		}
	}
}

func processFile(filePath string) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("File does not exist: %s", filePath)
		return
	}

	htmlBytes, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading html file %s: %v", filePath, err)
		return
	}

	// Extract text content from HTML
	rawTextFile := parseHTML(&htmlBytes)
	if rawTextFile == "" {
		log.Printf("No text content extracted from %s", filePath)
		return
	}

	// Tokenize the extracted text
	tokens := Tokenize(rawTextFile)
	if len(tokens) == 0 {
		log.Printf("No tokens generated from %s", filePath)
		return
	}

	// Extract document ID from filename
	docID := filepath.Base(filePath)
	docID = strings.TrimSuffix(docID, ".html")

	// Add document to inverted index
	InvertedIndex.AddDocument(docID, tokens)

	log.Printf("Successfully processed %s: %d tokens indexed", docID, len(tokens))
}

// ShutdownExtractor gracefully shuts down the text extractor
func ShutdownExtractor() {
	log.Println("Initiating text extractor shutdown...")
	extractorCancel()

	// Close the channel to signal no more work
	close(IndexTargetChan)

	// Wait for all processing to complete
	extractorWg.Wait()
	log.Println("Text extractor shutdown complete")
}

// GetExtractorStats returns statistics about the text extractor
func GetExtractorStats() map[string]interface{} {
	return map[string]interface{}{
		"queue_size":     len(IndexTargetChan),
		"queue_capacity": cap(IndexTargetChan),
		"active_workers": extractorWg,
	}
}

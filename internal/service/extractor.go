package service

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

// TODO concurrently parse and extract tokens from our html byte streams,
// We traverse these documents using our traverseDOMTree function to extract all
// relevant tokens then we pass it on to the indexer which builds the term frequency inverted
// Document frequency matrix (TF-IDF) [CURRENT IMPLEMENTATION SUBJECT TO CHANGE]

var indexTargetChan = make(chan string, 10)

func parseHTML(htmlBytes *[]byte) string {
	var rawText strings.Builder
	htmlReader := bytes.NewReader(*htmlBytes)
	node, err := html.Parse(htmlReader)
	if err != nil {
		log.Printf("error parsing html : %q\n", err)
	}

	visit := func(node *html.Node) {
		if node.Type == html.TextNode {
			rawText.WriteString(node.Data)
		}
	}
	traverseDOMTree(node, visit)
	return rawText.String()
}

func ExtractText() string {
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for filePath := range indexTargetChan {
		sem <- struct{}{}
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			defer func() { <-sem }()
			htmlBytes, err := os.ReadFile(filePath)
			if err != nil {
				log.Printf("error parsing html file : %q", err)
				return
			}
			rawTextFile := parseHTML(&htmlBytes)
			fmt.Println(rawTextFile)
		}(filePath)
	}
	wg.Wait()

	return ""
}

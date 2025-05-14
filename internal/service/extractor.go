package service

import "sync"

var indexTargetChan = make(chan string, 10)

// TODO concurrently parse and extract tokens from our html byte streams,
// We traverse these documents using our traverseDOMTree function to extract all
// relevant tokens then we pass it on to the indexer which builds the term frequency inverted
// Document frequency matrix (TF-IDF) [CURRENT IMPLEMENTATION SUBJECT TO CHANGE]

func textExtractionWorker(filePath string, sem chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	sem <- struct{}{}
	defer func() { <-sem }()
}

func ExtractText() string {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for filePath := range indexTargetChan {
		wg.Add(1)
		go textExtractionWorker(filePath, sem, &wg)
	}
	wg.Wait()
	return ""
}

package service

import (
	"log"
	"sync"
)

type Index struct {
	mu    sync.RWMutex
	index map[string]map[string]int64
}

func (i *Index) addDocument(docID string, terms []string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	// TODO populate our index
	log.Println(docID, " : ", terms)
}

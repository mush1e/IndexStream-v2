package service

import (
	"fmt"
	"sync"
)

var InvertedIndex = NewInvertedIndex()

type Index struct {
	mu        sync.RWMutex
	index     map[string]map[string][]int
	docLen    map[string]int
	docFreq   map[string]int
	docCount  int
	avgDL     float64
	sumDocLen int
}

func NewInvertedIndex() *Index {
	return &Index{
		index:   make(map[string]map[string][]int),
		docLen:  make(map[string]int),
		docFreq: make(map[string]int),
	}
}

func (idx *Index) AddDocument(docID string, tokens []string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// If document with docID already exists, do nothing
	if _, found := idx.docLen[docID]; found {
		return
	}

	idx.docCount++
	idx.docLen[docID] = len(tokens)

	// track which term we've already bumped document frequency for
	seen := map[string]bool{}

	for pos, token := range tokens {
		// Init new term in index if needed
		if idx.index[token] == nil {
			idx.index[token] = make(map[string][]int)
		}
		idx.index[token][docID] = append(idx.index[token][docID], pos)

		// bump doc frequency once for each term
		if !seen[token] {
			idx.docFreq[token]++
			seen[token] = true
		}
	}

	// recompute avg doc length
	idx.sumDocLen += len(tokens)
	idx.avgDL = float64(idx.sumDocLen) / float64(idx.docCount)
	fmt.Printf("%q - %+v\n\n", docID, idx.docLen[docID])
}

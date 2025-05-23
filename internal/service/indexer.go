package service

import (
	"fmt"
	"math"
	"sort"
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

type SearchResult struct {
	DocID string
	Score float64
}

func (idx *Index) Search(query string, topK int) []SearchResult {
	terms := Tokenize(query)
	terms = tokenDeduper(terms)

	idx.mu.Lock()
	defer idx.mu.Unlock()

	candidates := map[string]struct{}{}

	for _, term := range terms {
		postings, ok := idx.index[term]
		if !ok {
			continue // term not in any doc
		}
		for docID := range postings {
			candidates[docID] = struct{}{}
		}
	}

	// Using BM25 to calculate term relevance and order search results
	scores := map[string]float64{}
	N := float64(idx.docCount)
	k1, b := 1.5, 0.75
	for _, term := range terms {
		postings := idx.index[term]      // map[docID][]positions
		df := float64(idx.docFreq[term]) // number of docs containing term
		if df == 0 {
			continue
		}
		idf := math.Log((N - df + 0.5) / (df + 0.5)) // BM25 IDF

		for docID := range candidates {
			positions := postings[docID] // might be nil or empty
			tf := float64(len(positions))
			if tf == 0 {
				continue
			}
			dl := float64(idx.docLen[docID])
			avgdl := idx.avgDL
			scoreTerm := idf * (tf * (k1 + 1)) / (tf + k1*(1-b+b*(dl/avgdl)))
			scores[docID] += scoreTerm
		}
	}

	results := make([]SearchResult, 0, len(scores))
	for docID, score := range scores {
		results = append(results, SearchResult{DocID: DocURLMap[docID], Score: score})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Trim to topK
	if len(results) > topK {
		results = results[:topK]
	}
	return results
}

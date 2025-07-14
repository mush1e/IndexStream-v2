package service

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mush1e/IndexStream-v2/internal/cache"
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

	// Multi-layer cache
	cache *cache.MultiLayerCache

	// Document metadata cache
	docMetaCache map[string]*DocumentMetadata
	docMetaMutex sync.RWMutex
}

type DocumentMetadata struct {
	URL        string    `json:"url"`
	Title      string    `json:"title"`
	Length     int       `json:"length"`
	IndexedAt  time.Time `json:"indexed_at"`
	LastAccess time.Time `json:"last_access"`
}

type SearchResult struct {
	DocID    string            `json:"doc_id"`
	URL      string            `json:"url"`
	Title    string            `json:"title"`
	Score    float64           `json:"score"`
	Metadata *DocumentMetadata `json:"metadata,omitempty"`
}

type CachedSearchResults struct {
	Results   []SearchResult `json:"results"`
	Query     string         `json:"query"`
	Timestamp time.Time      `json:"timestamp"`
	TotalDocs int            `json:"total_docs"`
}

func NewInvertedIndex() *Index {
	cacheConfig := cache.DefaultCacheConfig()
	// Customize cache settings for search engine
	cacheConfig.L1MaxItems = 2000 // More items in memory
	cacheConfig.L1TTL = 1 * time.Hour
	cacheConfig.L2MaxSizeMB = 1000       // 1GB disk cache
	cacheConfig.L3TTL = 30 * time.Minute // Query results cache

	multiCache, err := cache.NewMultiLayerCache(cacheConfig)
	if err != nil {
		fmt.Printf("Failed to initialize cache: %v\n", err)
	}

	return &Index{
		index:        make(map[string]map[string][]int),
		docLen:       make(map[string]int),
		docFreq:      make(map[string]int),
		cache:        multiCache,
		docMetaCache: make(map[string]*DocumentMetadata),
	}
}

func (idx *Index) AddDocument(docID string, tokens []string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Check if document already exists in cache
	cacheKey := "doc:" + docID
	if idx.cache != nil {
		if _, exists := idx.cache.Get(cacheKey); exists {
			fmt.Printf("Document %q already cached, skipping...\n", docID)
			return
		}
	}

	// If document with docID already exists, do nothing
	if _, found := idx.docLen[docID]; found {
		return
	}

	idx.docCount++
	idx.docLen[docID] = len(tokens)

	// Track which terms we've already bumped document frequency for
	seen := map[string]bool{}

	for pos, token := range tokens {
		// Init new term in index if needed
		if idx.index[token] == nil {
			idx.index[token] = make(map[string][]int)
		}
		idx.index[token][docID] = append(idx.index[token][docID], pos)

		// Bump doc frequency once for each term
		if !seen[token] {
			idx.docFreq[token]++
			seen[token] = true
		}
	}

	// Recompute avg doc length
	idx.sumDocLen += len(tokens)
	idx.avgDL = float64(idx.sumDocLen) / float64(idx.docCount)

	// Create document metadata
	docURLMu.RLock()
	url := DocURLMap[docID]
	docURLMu.RUnlock()

	metadata := &DocumentMetadata{
		URL:        url,
		Title:      extractTitleFromURL(url),
		Length:     len(tokens),
		IndexedAt:  time.Now(),
		LastAccess: time.Now(),
	}

	// Cache document metadata
	idx.docMetaMutex.Lock()
	idx.docMetaCache[docID] = metadata
	idx.docMetaMutex.Unlock()

	// Cache the document in multi-layer cache
	if idx.cache != nil {
		docData := map[string]interface{}{
			"tokens":     tokens,
			"metadata":   metadata,
			"indexed_at": time.Now(),
		}
		idx.cache.Set(cacheKey, docData)
	}

	fmt.Printf("Document %q indexed (%d tokens) and cached\n", docID, len(tokens))
}

func (idx *Index) Search(query string, topK int) []SearchResult {
	start := time.Now()

	// Check query result cache first
	if idx.cache != nil {
		if cached, found := idx.cache.GetQueryResult(query); found {
			if cachedResults, ok := cached.(CachedSearchResults); ok {
				// Update access times for returned documents
				for _, result := range cachedResults.Results {
					idx.updateDocumentAccess(result.DocID)
				}
				fmt.Printf("Cache hit for query %q (%.2fms)\n", query,
					float64(time.Since(start).Nanoseconds())/1e6)
				return cachedResults.Results
			}
		}
	}

	// Cache miss - perform actual search
	results := idx.performSearch(query, topK)

	// Cache the results
	if idx.cache != nil {
		cachedResults := CachedSearchResults{
			Results:   results,
			Query:     query,
			Timestamp: time.Now(),
			TotalDocs: idx.docCount,
		}
		idx.cache.SetQueryResult(query, cachedResults)
	}

	searchTime := time.Since(start)
	fmt.Printf("Search completed for %q: %d results (%.2fms)\n",
		query, len(results), float64(searchTime.Nanoseconds())/1e6)

	return results
}

func (idx *Index) performSearch(query string, topK int) []SearchResult {
	terms := Tokenize(query)
	terms = tokenDeduper(terms)

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	candidates := map[string]struct{}{}

	// Check term cache first
	for _, term := range terms {
		cacheKey := "term:" + term
		var postings map[string][]int
		var found bool

		if idx.cache != nil {
			if cached, exists := idx.cache.Get(cacheKey); exists {
				if p, ok := cached.(map[string][]int); ok {
					postings = p
					found = true
				}
			}
		}

		if !found {
			// Term not in cache, get from index
			var ok bool
			postings, ok = idx.index[term]
			if !ok {
				continue // Term not in any doc
			}

			// Cache the term postings
			if idx.cache != nil {
				idx.cache.Set(cacheKey, postings)
			}
		}

		for docID := range postings {
			candidates[docID] = struct{}{}
		}
	}

	// Calculate BM25 scores
	scores := map[string]float64{}
	N := float64(idx.docCount)
	k1, b := 1.5, 0.75

	for _, term := range terms {
		postings := idx.index[term]
		df := float64(idx.docFreq[term])
		if df == 0 {
			continue
		}
		idf := math.Log((N - df + 0.5) / (df + 0.5)) // BM25 IDF

		for docID := range candidates {
			positions := postings[docID]
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
		// Get document metadata
		metadata := idx.getDocumentMetadata(docID)

		docURLMu.RLock()
		url := DocURLMap[docID]
		docURLMu.RUnlock()

		result := SearchResult{
			DocID:    docID,
			URL:      url,
			Title:    metadata.Title,
			Score:    score,
			Metadata: metadata,
		}
		results = append(results, result)

		// Update document access time
		idx.updateDocumentAccess(docID)
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Trim to topK
	if len(results) > topK {
		results = results[:topK]
	}

	return results
}

func (idx *Index) getDocumentMetadata(docID string) *DocumentMetadata {
	idx.docMetaMutex.RLock()
	defer idx.docMetaMutex.RUnlock()

	if metadata, exists := idx.docMetaCache[docID]; exists {
		return metadata
	}

	// Fallback metadata if not found
	docURLMu.RLock()
	url := DocURLMap[docID]
	docURLMu.RUnlock()

	return &DocumentMetadata{
		URL:        url,
		Title:      extractTitleFromURL(url),
		Length:     idx.docLen[docID],
		IndexedAt:  time.Now(),
		LastAccess: time.Now(),
	}
}

func (idx *Index) updateDocumentAccess(docID string) {
	idx.docMetaMutex.Lock()
	defer idx.docMetaMutex.Unlock()

	if metadata, exists := idx.docMetaCache[docID]; exists {
		metadata.LastAccess = time.Now()
	}
}

func extractTitleFromURL(url string) string {
	if url == "" {
		return "Untitled Document"
	}

	// Simple title extraction from URL
	// In a real implementation, you'd parse the HTML <title> tag
	parts := strings.Split(url, "/")
	if len(parts) > 2 {
		domain := parts[2]
		if len(parts) > 3 {
			return fmt.Sprintf("%s - %s", domain, parts[len(parts)-1])
		}
		return domain
	}
	return url
}

// GetCacheStats returns cache performance statistics
func (idx *Index) GetCacheStats() map[string]interface{} {
	if idx.cache == nil {
		return map[string]interface{}{"error": "cache not initialized"}
	}
	return idx.cache.GetCacheInfo()
}

// ClearCache clears all cache layers
func (idx *Index) ClearCache() error {
	if idx.cache == nil {
		return fmt.Errorf("cache not initialized")
	}
	return idx.cache.Clear()
}

// PrewarmCache loads frequently accessed terms into cache
func (idx *Index) PrewarmCache() {
	if idx.cache == nil {
		return
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	fmt.Println("Prewarming cache with frequently used terms...")

	// Create a frequency map of terms
	termFreq := make(map[string]int)
	for term, docs := range idx.index {
		termFreq[term] = len(docs)
	}

	// Sort terms by frequency
	type termFreqPair struct {
		term string
		freq int
	}

	var termFreqList []termFreqPair
	for term, freq := range termFreq {
		termFreqList = append(termFreqList, termFreqPair{term, freq})
	}

	sort.Slice(termFreqList, func(i, j int) bool {
		return termFreqList[i].freq > termFreqList[j].freq
	})

	// Cache top 500 most frequent terms
	limit := 500
	if len(termFreqList) < limit {
		limit = len(termFreqList)
	}

	for i := 0; i < limit; i++ {
		term := termFreqList[i].term
		cacheKey := "term:" + term
		postings := idx.index[term]
		idx.cache.Set(cacheKey, postings)
	}

	fmt.Printf("Prewarmed cache with %d frequent terms\n", limit)
}

// OptimizeCache performs maintenance operations on the cache
func (idx *Index) OptimizeCache() {
	if idx.cache == nil {
		return
	}

	fmt.Println("Optimizing cache...")

	// Get current stats
	stats := idx.cache.GetStats()

	// If hit rate is low, clear some cache to make room for better data
	totalL1Requests := stats.L1Hits + stats.L1Misses
	if totalL1Requests > 100 {
		l1HitRate := float64(stats.L1Hits) / float64(totalL1Requests)
		if l1HitRate < 0.3 { // Less than 30% hit rate
			fmt.Printf("Low L1 hit rate (%.2f%%), clearing cache for optimization\n", l1HitRate*100)
			idx.cache.Clear()
			// Rewarm with current high-frequency terms
			idx.PrewarmCache()
		}
	}
}

// InvalidateDocument removes a document from all caches
func (idx *Index) InvalidateDocument(docID string) {
	// This would be called when a document is updated or deleted
	// cacheKey := "doc:" + docID

	// Remove from metadata cache
	idx.docMetaMutex.Lock()
	delete(idx.docMetaCache, docID)
	idx.docMetaMutex.Unlock()

	// Note: In a full implementation, you'd also need to remove the document
	// from the main index and update term frequencies
	fmt.Printf("Invalidated caches for document %q\n", docID)
}

// GetDocumentCount returns the total number of indexed documents
func (idx *Index) GetDocumentCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.docCount
}

// GetIndexStats returns statistics about the inverted index
func (idx *Index) GetIndexStats() map[string]interface{} {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	uniqueTerms := len(idx.index)
	totalPositions := 0

	for _, termDocs := range idx.index {
		for _, positions := range termDocs {
			totalPositions += len(positions)
		}
	}

	return map[string]interface{}{
		"total_documents":    idx.docCount,
		"unique_terms":       uniqueTerms,
		"total_positions":    totalPositions,
		"average_doc_length": idx.avgDL,
		"cache_stats":        idx.GetCacheStats(),
	}
}

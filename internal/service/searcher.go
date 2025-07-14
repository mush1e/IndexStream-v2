package service

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// SearchEngine wraps the inverted index with additional search functionality
type SearchEngine struct {
	index *Index
	mu    sync.RWMutex
}

// NewSearchEngine creates a new search engine instance
func NewSearchEngine() *SearchEngine {
	return &SearchEngine{
		index: InvertedIndex,
	}
}

// SearchOptions provides configuration for search operations
type SearchOptions struct {
	MaxResults    int
	MinScore      float64
	BoostExact    bool
	CaseSensitive bool
}

// DefaultSearchOptions returns sensible default search options
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		MaxResults:    10,
		MinScore:      0.0,
		BoostExact:    true,
		CaseSensitive: false,
	}
}

// EnhancedSearchResult extends SearchResult with additional metadata
type EnhancedSearchResult struct {
	DocID        string           `json:"doc_id"`
	URL          string           `json:"url"`
	Score        float64          `json:"score"`
	MatchedTerms []string         `json:"matched_terms"`
	Snippet      string           `json:"snippet,omitempty"`
	Highlights   map[string][]int `json:"highlights,omitempty"`
}

// PerformSearch executes a search with enhanced options and returns detailed results
func (se *SearchEngine) PerformSearch(query string, options SearchOptions) ([]EnhancedSearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return []EnhancedSearchResult{}, fmt.Errorf("empty search query")
	}

	se.mu.RLock()
	defer se.mu.RUnlock()

	// Tokenize and prepare search terms
	terms := Tokenize(query)
	if len(terms) == 0 {
		return []EnhancedSearchResult{}, fmt.Errorf("no valid search terms found")
	}

	// Remove duplicates
	terms = tokenDeduper(terms)

	// Get basic search results
	basicResults := se.index.Search(query, options.MaxResults*2) // Get more to filter later

	// Convert to enhanced results
	enhancedResults := make([]EnhancedSearchResult, 0, len(basicResults))

	for _, result := range basicResults {
		if result.Score < options.MinScore {
			continue
		}

		docURLMu.RLock()
		url, exists := DocURLMap[result.DocID]
		docURLMu.RUnlock()

		if !exists {
			url = result.DocID // fallback to docID if URL not found
		}

		// Find matched terms for this document
		matchedTerms := se.findMatchedTerms(result.DocID, terms)

		enhancedResult := EnhancedSearchResult{
			DocID:        result.DocID,
			URL:          url,
			Score:        result.Score,
			MatchedTerms: matchedTerms,
		}

		// Apply exact match boosting if enabled
		if options.BoostExact && se.hasExactMatch(result.DocID, query) {
			enhancedResult.Score *= 1.5
		}

		enhancedResults = append(enhancedResults, enhancedResult)
	}

	// Re-sort after potential score boosting
	sort.Slice(enhancedResults, func(i, j int) bool {
		return enhancedResults[i].Score > enhancedResults[j].Score
	})

	// Trim to requested max results
	if len(enhancedResults) > options.MaxResults {
		enhancedResults = enhancedResults[:options.MaxResults]
	}

	return enhancedResults, nil
}

// findMatchedTerms identifies which search terms were found in a document
func (se *SearchEngine) findMatchedTerms(docID string, searchTerms []string) []string {
	var matched []string

	for _, term := range searchTerms {
		if postings, exists := se.index.index[term]; exists {
			if _, found := postings[docID]; found {
				matched = append(matched, term)
			}
		}
	}

	return matched
}

// hasExactMatch checks if the document contains the exact query phrase
func (se *SearchEngine) hasExactMatch(docID, query string) bool {
	// This is a simplified implementation
	// In a real system, you'd want to check for exact phrase matches
	queryTerms := Tokenize(query)
	if len(queryTerms) <= 1 {
		return false
	}

	// Check if all consecutive terms appear in the document
	foundCount := 0
	for _, term := range queryTerms {
		if postings, exists := se.index.index[term]; exists {
			if _, found := postings[docID]; found {
				foundCount++
			}
		}
	}

	return foundCount == len(queryTerms)
}

// GetDocumentStats returns statistics about a specific document
func (se *SearchEngine) GetDocumentStats(docID string) map[string]interface{} {
	se.mu.RLock()
	defer se.mu.RUnlock()

	stats := make(map[string]interface{})

	if docLen, exists := se.index.docLen[docID]; exists {
		stats["document_length"] = docLen
		stats["exists"] = true

		// Count unique terms in document
		uniqueTerms := 0
		for _, postings := range se.index.index {
			if _, found := postings[docID]; found {
				uniqueTerms++
			}
		}
		stats["unique_terms"] = uniqueTerms

		docURLMu.RLock()
		if url, exists := DocURLMap[docID]; exists {
			stats["url"] = url
		}
		docURLMu.RUnlock()
	} else {
		stats["exists"] = false
	}

	return stats
}

// GetIndexStats returns overall statistics about the search index
func (se *SearchEngine) GetIndexStats() map[string]interface{} {
	se.mu.RLock()
	defer se.mu.RUnlock()

	stats := map[string]interface{}{
		"total_documents":    se.index.docCount,
		"total_terms":        len(se.index.index),
		"average_doc_length": se.index.avgDL,
		"total_doc_length":   se.index.sumDocLen,
	}

	return stats
}

// SuggestSimilar provides basic query suggestions based on indexed terms
func (se *SearchEngine) SuggestSimilar(partialQuery string, maxSuggestions int) []string {
	se.mu.RLock()
	defer se.mu.RUnlock()

	if maxSuggestions <= 0 {
		maxSuggestions = 5
	}

	partialQuery = strings.ToLower(strings.TrimSpace(partialQuery))
	if len(partialQuery) < 2 {
		return []string{}
	}

	var suggestions []string

	for term := range se.index.index {
		if strings.HasPrefix(term, partialQuery) && term != partialQuery {
			suggestions = append(suggestions, term)
			if len(suggestions) >= maxSuggestions {
				break
			}
		}
	}

	// Sort suggestions by document frequency (more common terms first)
	sort.Slice(suggestions, func(i, j int) bool {
		freqI := se.index.docFreq[suggestions[i]]
		freqJ := se.index.docFreq[suggestions[j]]
		return freqI > freqJ
	})

	return suggestions
}

// SearchByTerms performs a search using pre-tokenized terms
func (se *SearchEngine) SearchByTerms(terms []string, maxResults int) []SearchResult {
	if len(terms) == 0 {
		return []SearchResult{}
	}

	// Join terms back into a query string for the existing search method
	query := strings.Join(terms, " ")
	return se.index.Search(query, maxResults)
}

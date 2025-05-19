package service

type SearchResult struct {
	DocID string
	Score float64
}

func Search(query string) []SearchResult {
	return nil
}

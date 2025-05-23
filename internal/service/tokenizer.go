package service

import (
	"strings"
	"unicode"

	"github.com/kljensen/snowball"
)

// Helper to stem words eg. running -> run
func wordStemmer(word string) (string, error) {
	return snowball.Stem(word, "english", true)
}

func tokenDeduper(tokens []string) []string {
	hashSet := make(map[string]struct{})
	dedupedTokenList := make([]string, 0, len(tokens))
	for _, token := range tokens {
		hashSet[token] = struct{}{}
	}

	for token := range hashSet {
		dedupedTokenList = append(dedupedTokenList, token)
	}
	return dedupedTokenList
}

func Tokenize(doc string) []string {
	doc = strings.ToLower(doc)
	var cleanedDoc strings.Builder

	for _, r := range doc {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			cleanedDoc.WriteRune(r)
		}
	}

	rawTokens := strings.Fields(cleanedDoc.String())
	tokens := make([]string, len(rawTokens))
	for idx, token := range rawTokens {
		stemmedToken, err := wordStemmer(token)
		if err != nil {
			stemmedToken = token
		}
		tokens[idx] = stemmedToken
	}
	return tokens
}

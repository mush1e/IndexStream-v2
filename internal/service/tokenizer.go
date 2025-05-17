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

package service

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Helper function to check if URL is valid HTTP(S)
func isValidHTTPURL(u *url.URL) bool {
	return u.Host != "" && (u.Scheme == "https" || u.Scheme == "http")
}

func preprocessRawURL(rawURL, baseURL string) string {
	if rawURL == "" {
		return ""
	}

	rawURLObj, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	// Return direct absolute URLs that use http(s)
	if rawURLObj.IsAbs() {
		if isValidHTTPURL(rawURLObj) {
			return rawURLObj.String()
		}
		return ""
	}

	// Resolve relative URLs against base URL
	baseURLObj, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	resolvedURL := baseURLObj.ResolveReference(rawURLObj)
	if isValidHTTPURL(resolvedURL) {
		return resolvedURL.String()
	}

	return ""
}

// Pre-Order Traversal on DOM Tree
func traverseDOMTree(node *html.Node, visit func(*html.Node)) {
	visit(node)
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		traverseDOMTree(n, visit)
	}
}

func URLExtractor(fileContent []byte, baseURL string) (map[string]struct{}, error) {
	reader := strings.NewReader(string(fileContent))
	htmlRoot, err := html.Parse(reader)

	if err != nil {
		return nil, fmt.Errorf("parsing HTML of %s: %w", baseURL, err)
	}

	extractedURLs := make(map[string]struct{})

	// Callback function to extract links from A tags
	visitDOMElement := func(node *html.Node) {
		if node.Type == html.ElementNode && node.DataAtom == atom.A {
			for _, attr := range node.Attr {
				if attr.Key == "href" {
					rawURL := attr.Val
					url := preprocessRawURL(rawURL, baseURL)
					extractedURLs[url] = struct{}{}
				}
			}
		}
	}

	traverseDOMTree(htmlRoot, visitDOMElement)
	return extractedURLs, nil
}

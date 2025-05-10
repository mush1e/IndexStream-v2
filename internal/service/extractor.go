package service

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func preprocessRawURL(rawUrl string) string {
	return rawUrl
}

func traverseDOMTree(node *html.Node, visit func(*html.Node)) {
	visit(node)
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		traverseDOMTree(n, visit)
	}
}

func URLExtractor(fileContent []byte, baseUrl string) (map[string]struct{}, error) {
	reader := strings.NewReader(string(fileContent))
	htmlRoot, err := html.Parse(reader)

	if err != nil {
		return nil, err
	}

	extractedURLs := make(map[string]struct{})

	visitDOMElement := func(node *html.Node) {
		if node.Type == html.ElementNode && node.DataAtom == atom.A {
			for _, attr := range node.Attr {
				if attr.Key == "href" {
					rawURL := attr.Val
					url := preprocessRawURL(rawURL)
					extractedURLs[url] = struct{}{}
				}
			}
		}
	}

	traverseDOMTree(htmlRoot, visitDOMElement)
	return extractedURLs, nil
}

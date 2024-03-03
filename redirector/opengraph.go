package redirector

import (
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

// Function to get all Open Graph meta tags from a given URL as a string
func getOpenGraphTagsHTML(url string) (string, error) {
	// Make HTTP GET request to the URL
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Parse HTML content of the response body
	tokenizer := html.NewTokenizer(resp.Body)

	// Initialize string to store HTML meta tags
	var ogTagsHTML strings.Builder

	// Loop through tokens to find meta tags
	for {
		// Get next token type
		tokenType := tokenizer.Next()

		// If token type is an error or end of document, break loop
		if tokenType == html.ErrorToken {
			break
		}

		// If token type is a start tag
		if tokenType == html.StartTagToken {
			// Get token
			token := tokenizer.Token()

			// If token is a meta tag
			if token.Data == "meta" {
				// Get attributes of meta tag
				for _, attr := range token.Attr {
					// If attribute is an Open Graph property
					if attr.Key == "property" && attr.Val[:3] == "og:" {
						// Append HTML meta tag to string
						ogTagsHTML.WriteString(fmt.Sprintf(`<meta property="%s" content="%s">`, attr.Val, token.Attr[1].Val))
						ogTagsHTML.WriteString("\n")
					}
				}
			}
		}
	}

	return ogTagsHTML.String(), nil
}

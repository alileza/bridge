package redirector

import (
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// FetchHeadElements fetches all HTML elements inside <head> from the given target URL
func FetchHeadElements(targetURL string) (string, error) {
	// Function to fetch HTML content from URL
	getHTMLFromURL := func(url string) (*html.Node, error) {
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		return html.Parse(resp.Body)
	}

	// Recursive function to find all elements within <head> excluding scripts, stylesheets, and style tags
	findHeadElements := func(n *html.Node) []*html.Node {
		var headElements []*html.Node
		var traverse func(*html.Node)
		traverse = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "head" {
				for child := n.FirstChild; child != nil; child = child.NextSibling {
					// Exclude script, link, and style tags
					if child.Data != "script" && child.Data != "link" && child.Data != "style" {
						headElements = append(headElements, child)
					}
				}
				return
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				traverse(c)
			}
		}
		traverse(n)
		return headElements
	}

	// Extract head elements from URL
	root, err := getHTMLFromURL(targetURL)
	if err != nil {
		return "", err
	}
	headElements := findHeadElements(root)

	// Render head elements as strings
	var result strings.Builder
	for _, elem := range headElements {
		html.Render(&result, elem)
	}

	return result.String(), nil
}

// MinifyHTML minifies the given HTML string
func MinifyHTML(htmlString string) string {
	// Remove extra whitespace characters
	htmlString = strings.TrimSpace(htmlString)
	htmlString = strings.ReplaceAll(htmlString, "\n", "")
	htmlString = strings.ReplaceAll(htmlString, "\t", "")
	htmlString = strings.ReplaceAll(htmlString, "\r", "")

	// Remove HTML comments
	re := regexp.MustCompile(`<!--(.*?)-->`)
	htmlString = re.ReplaceAllString(htmlString, "")

	// Remove spaces around tags
	htmlString = regexp.MustCompile(`>\s+<`).ReplaceAllString(htmlString, "><")

	return htmlString
}

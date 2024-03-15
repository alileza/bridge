package opengraph

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/net/html"
)

type OpenGraph struct {
	Client *http.Client
}

// FetchMetaTags fetches meta tags from the head section of a given URL
func (og *OpenGraph) FetchMetaTags(ctx context.Context, targetURL string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := og.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	var metaTags []string
	tokenizer := html.NewTokenizer(resp.Body)
	var inHead bool
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			err := tokenizer.Err()
			if err == io.EOF {
				return metaTags, nil // EOF is expected
			}
			return nil, fmt.Errorf("reading body: %w", err)
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			if token.Data == "head" {
				inHead = true
			}
			if inHead && token.Data == "meta" {
				metaTags = append(metaTags, token.String())
			}
		case html.EndTagToken:
			token := tokenizer.Token()
			if inHead && token.Data == "head" {
				return metaTags, nil // End of head section
			}
		}
	}
}

func FetchMetaTags(ctx context.Context, targetURL string) ([]string, error) {
	og := OpenGraph{Client: http.DefaultClient}
	return og.FetchMetaTags(ctx, targetURL)
}

package redirector

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetOpenGraphTagsHTML(t *testing.T) {
	// Create a test server to serve a mock HTML page
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		htmlContent := `
		<!DOCTYPE html>
		<html>
		<head>
			<meta property="og:title" content="Test Title">
			<meta property="og:description" content="Test Description">
			<meta property="og:url" content="https://example.com">
		</head>
		<body>
		</body>
		</html>
		`
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(htmlContent))
	}))
	defer ts.Close()

	// Call the function with the test server URL
	ogTagsHTML, err := getOpenGraphTagsHTML(ts.URL)
	if err != nil {
		t.Errorf("Error fetching Open Graph tags HTML: %v", err)
	}

	log.Printf("Open Graph tags HTML: %s", ogTagsHTML)

	// Expected Open Graph tags HTML
	expected := `<meta property="og:title" content="Test Title">
<meta property="og:description" content="Test Description">
<meta property="og:url" content="https://example.com">
`

	// Compare the actual and expected Open Graph tags HTML
	if ogTagsHTML != expected {
		t.Errorf("Unexpected Open Graph tags HTML. Got: %s, Expected: %s", ogTagsHTML, expected)
	}
}

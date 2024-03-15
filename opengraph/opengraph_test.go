package opengraph

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestFetchMetaTags(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantTags   []string
		wantErrMsg string
	}{
		{
			name: "Valid URL with meta tags",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`
					<!DOCTYPE html>
					<html>
					<head>
						<meta name="description" content="An example website">
						<meta name="keywords" content="example, website">
					</head>
					<body>
					</body>
					</html>
				`))
			},
			wantTags: []string{
				`<meta name="description" content="An example website">`,
				`<meta name="keywords" content="example, website">`,
			},
			wantErrMsg: "",
		},
		{
			name: "Invalid URL",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantTags:   nil,
			wantErrMsg: "HTTP error: 404 Not Found",
		},
		{
			name: "Timeout when fetching URL",
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(2 * time.Second)
			},
			wantTags:   nil,
			wantErrMsg: "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			ctx := context.Background()
			gotTags, err := FetchMetaTags(ctx, server.URL)

			if err != nil {
				if err.Error() != tt.wantErrMsg {
					t.Errorf("FetchMetaTags() error = %v, wantErr %v", err, tt.wantErrMsg)
					return
				}
			} else {
				if !reflect.DeepEqual(gotTags, tt.wantTags) {
					t.Errorf("FetchMetaTags() = %v, want %v", gotTags, tt.wantTags)
				}
			}
		})
	}

}

package httpredirector

import (
	"reflect"
	"testing"
)

func TestFilterRoutes(t *testing.T) {
	routes := []Route{
		{Key: "go.example.com/gh", URL: "https://github.com"},
		{Key: "go.example.com/docs", URL: "https://docs.example.com"},
		{Key: "go.example.com.evil/gh", URL: "https://evil.example"},
		{Key: "other.example.com/gh", URL: "https://github.com"},
	}

	got := filterRoutes(routes, "go.example.com")
	want := []Route{routes[0], routes[1]}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("filterRoutes() = %v, want %v", got, want)
	}

	if got := filterRoutes(routes, "none.example.com"); got == nil || len(got) != 0 {
		t.Errorf("filterRoutes() with no match = %#v, want empty non-nil slice", got)
	}
}

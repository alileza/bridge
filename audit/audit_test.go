package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "routes.audit.jsonl")

	l, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	l.Record(Entry{Actor: "alice", Action: ActionCreate, Key: "go/a", URL: "https://a"})
	l.Record(Entry{Actor: "bob", Action: ActionCreate, Key: "go/b", URL: "https://b"})
	l.Record(Entry{Actor: "carol", Action: ActionUpdate, Key: "go/a", URL: "https://a2", PreviousURL: "https://a"})
	l.Close()

	// A torn trailing line must not stop the log from loading.
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString(`{"time":"2026-01-0`)
	f.Close()

	l, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if got := l.Recent(10, ""); len(got) != 3 || got[0].Actor != "carol" || got[2].Actor != "alice" {
		t.Errorf("Recent(10) = %+v, want 3 entries newest first", got)
	}
	if got := l.Recent(1, ""); len(got) != 1 {
		t.Errorf("Recent(1) returned %d entries", len(got))
	}
	if got := l.Recent(10, "go/a"); len(got) != 2 {
		t.Errorf("Recent(key) returned %d entries, want 2", len(got))
	}
	if e, ok := l.Last("go/a"); !ok || e.Actor != "carol" || e.PreviousURL != "https://a" {
		t.Errorf("Last(go/a) = %+v, %v", e, ok)
	}
	if e, ok := l.Last("go/a"); !ok || e.Time.IsZero() {
		t.Errorf("Last(go/a) has zero time: %+v", e)
	}
	if _, ok := l.Last("go/missing"); ok {
		t.Error("Last(missing) reported a change")
	}
}

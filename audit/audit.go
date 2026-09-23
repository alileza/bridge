// Package audit keeps an append-only JSON Lines log of route changes.
package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

const (
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
)

type Entry struct {
	Time        time.Time `json:"time"`
	Actor       string    `json:"actor"`
	Action      string    `json:"action"`
	Key         string    `json:"key"`
	URL         string    `json:"url,omitempty"`
	PreviousURL string    `json:"previous_url,omitempty"`
}

// Log appends entries to a file and keeps them in memory for querying.
type Log struct {
	mu      sync.RWMutex
	f       *os.File
	entries []Entry
	last    map[string]Entry
}

// Open loads existing entries from path and opens it for appending.
func Open(path string) (*Log, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("audit: open %s: %w", path, err)
	}

	l := &Log{f: f, last: map[string]Entry{}}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		var e Entry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			continue // skip a torn or corrupt line rather than refusing to start
		}
		l.add(e)
	}
	if err := sc.Err(); err != nil {
		f.Close()
		return nil, fmt.Errorf("audit: read %s: %w", path, err)
	}
	return l, nil
}

// Record appends an entry, stamping its time when unset.
func (l *Log) Record(e Entry) error {
	if e.Time.IsZero() {
		e.Time = time.Now().UTC()
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := l.f.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("audit: write: %w", err)
	}
	l.add(e)
	return nil
}

// Recent returns up to limit entries, newest first, optionally for one key.
func (l *Log) Recent(limit int, key string) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := []Entry{}
	for i := len(l.entries) - 1; i >= 0 && len(out) < limit; i-- {
		if key == "" || l.entries[i].Key == key {
			out = append(out, l.entries[i])
		}
	}
	return out
}

// Last returns the most recent change to key.
func (l *Log) Last(key string) (Entry, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	e, ok := l.last[key]
	return e, ok
}

func (l *Log) Close() error {
	return l.f.Close()
}

func (l *Log) add(e Entry) {
	l.entries = append(l.entries, e)
	l.last[e.Key] = e
}

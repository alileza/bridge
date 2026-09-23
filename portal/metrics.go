package portal

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// counterVec is a minimal Prometheus-compatible counter with a single label.
type counterVec struct {
	name, help, label string

	mu     sync.Mutex
	values map[string]uint64
}

func newCounterVec(name, help, label string) *counterVec {
	return &counterVec{name: name, help: help, label: label, values: map[string]uint64{}}
}

func (c *counterVec) Inc(labelValue string) {
	c.mu.Lock()
	c.values[labelValue]++
	c.mu.Unlock()
}

// ServeHTTP writes the counter in the Prometheus text exposition format.
func (c *counterVec) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	c.mu.Lock()
	keys := make([]string, 0, len(c.values))
	for k := range c.values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	snapshot := make([]uint64, len(keys))
	for i, k := range keys {
		snapshot[i] = c.values[k]
	}
	c.mu.Unlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n", c.name, c.help, c.name)
	for i, k := range keys {
		fmt.Fprintf(w, "%s{%s=%s} %d\n", c.name, c.label, escapeLabelValue(k), snapshot[i])
	}
}

var labelEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)

// escapeLabelValue quotes a label value per the Prometheus text format
// (only backslash, double-quote and newline are escaped).
func escapeLabelValue(v string) string {
	return `"` + labelEscaper.Replace(v) + `"`
}

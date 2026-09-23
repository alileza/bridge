package portal

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alileza/bridge/httpredirector"
)

// Metrics records bridge activity and serves it in the Prometheus text
// exposition format, without pulling in the Prometheus client library.
type Metrics struct {
	version string
	start   time.Time
	storage httpredirector.Storage

	httpRequests  *counterVec
	httpDuration  *histogramVec
	redirects     *counterVec
	forwarded     *counterVec
	misses        *counterVec
	routeChanges  *counterVec
	storageErrors *counterVec
}

func NewMetrics(version string, storage httpredirector.Storage) *Metrics {
	return &Metrics{
		version: version,
		start:   time.Now(),
		storage: storage,

		httpRequests:  newCounterVec("bridge_http_requests_total", "HTTP requests handled, by route pattern and status code.", "handler", "code"),
		httpDuration:  newHistogramVec("bridge_http_request_duration_seconds", "HTTP request latency, by route pattern.", "handler"),
		redirects:     newCounterVec("bridge_redirects_total", "Short links resolved and redirected, by host.", "host"),
		forwarded:     newCounterVec("bridge_routes_forwarded_total", "Short links resolved and redirected, by key.", "key"),
		misses:        newCounterVec("bridge_redirect_misses_total", "Lookups that matched no short link, by host.", "host"),
		routeChanges:  newCounterVec("bridge_route_changes_total", "Routes created/updated or deleted, by operation.", "op"),
		storageErrors: newCounterVec("bridge_storage_errors_total", "Storage operations that failed, by operation.", "op"),
	}
}

// Instrument wraps a ServeMux-backed handler to record request count and latency.
func (m *Metrics) Instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rec, r)

		handler := r.Pattern
		if handler == "" {
			handler = "unmatched"
		}
		m.httpRequests.Inc(handler, strconv.Itoa(rec.code))
		m.httpDuration.Observe(time.Since(start).Seconds(), handler)
	})
}

// ServeHTTP writes all metrics in the Prometheus text exposition format.
func (m *Metrics) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	writeGauge(w, "bridge_build_info", "Build information; always 1.",
		sample{labels: []string{"version", m.version, "goversion", runtime.Version()}, value: 1})
	writeGauge(w, "bridge_start_time_seconds", "Unix time the server started.",
		sample{value: float64(m.start.Unix())})

	routes, err := m.storage.List()
	healthy := 1.0
	if err != nil {
		healthy = 0
	}
	writeGauge(w, "bridge_storage_up", "Whether the route storage is readable (1) or not (0).", sample{value: healthy})
	writeGauge(w, "bridge_routes", "Routes currently registered, by host.", routesByHost(routes)...)

	for _, c := range []*counterVec{m.httpRequests, m.redirects, m.forwarded, m.misses, m.routeChanges, m.storageErrors} {
		c.write(w)
	}
	m.httpDuration.write(w)

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	writeGauge(w, "go_goroutines", "Number of goroutines that currently exist.", sample{value: float64(runtime.NumGoroutine())})
	writeGauge(w, "go_memstats_heap_alloc_bytes", "Bytes of allocated heap objects.", sample{value: float64(ms.HeapAlloc)})
	writeGauge(w, "go_memstats_sys_bytes", "Bytes of memory obtained from the OS.", sample{value: float64(ms.Sys)})
	writeCounter(w, "go_gc_cycles_total", "Completed GC cycles.", sample{value: float64(ms.NumGC)})
}

func routesByHost(routes []httpredirector.Route) []sample {
	counts := map[string]int{}
	for _, r := range routes {
		host, _, _ := strings.Cut(r.Key, "/")
		counts[host]++
	}
	samples := make([]sample, 0, len(counts))
	for host, n := range counts {
		samples = append(samples, sample{labels: []string{"host", host}, value: float64(n)})
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i].labels[1] < samples[j].labels[1] })
	return samples
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

// sample is one metric value; labels alternate name, value, name, value...
type sample struct {
	labels []string
	value  float64
}

func writeGauge(w io.Writer, name, help string, samples ...sample) {
	writeFamily(w, name, help, "gauge", samples)
}

func writeCounter(w io.Writer, name, help string, samples ...sample) {
	writeFamily(w, name, help, "counter", samples)
}

func writeFamily(w io.Writer, name, help, typ string, samples []sample) {
	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s %s\n", name, help, name, typ)
	for _, s := range samples {
		fmt.Fprintf(w, "%s%s %s\n", name, formatLabels(s.labels), formatValue(s.value))
	}
}

func formatLabels(pairs []string) string {
	if len(pairs) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteByte('{')
	for i := 0; i+1 < len(pairs); i += 2 {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(pairs[i])
		b.WriteString(`="`)
		b.WriteString(labelEscaper.Replace(pairs[i+1]))
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String()
}

func formatValue(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}

// labelEscaper escapes label values per the Prometheus text format
// (only backslash, double-quote and newline are escaped).
var labelEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)

// counterVec is a counter partitioned by a fixed set of label names.
type counterVec struct {
	name, help string
	labels     []string

	mu     sync.Mutex
	values map[string]float64
}

func newCounterVec(name, help string, labels ...string) *counterVec {
	return &counterVec{name: name, help: help, labels: labels, values: map[string]float64{}}
}

func (c *counterVec) Inc(labelValues ...string) {
	key := strings.Join(labelValues, "\xff")
	c.mu.Lock()
	c.values[key]++
	c.mu.Unlock()
}

func (c *counterVec) write(w io.Writer) {
	c.mu.Lock()
	samples := make([]sample, 0, len(c.values))
	for key, v := range c.values {
		samples = append(samples, sample{labels: zipLabels(c.labels, strings.Split(key, "\xff")), value: v})
	}
	c.mu.Unlock()
	sortSamples(samples)
	writeCounter(w, c.name, c.help, samples...)
}

var defaultBuckets = []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}

// histogramVec is a histogram partitioned by a single label.
type histogramVec struct {
	name, help, label string

	mu     sync.Mutex
	series map[string]*histogram
}

type histogram struct {
	buckets []uint64 // cumulative counts are computed at write time
	sum     float64
	count   uint64
}

func newHistogramVec(name, help, label string) *histogramVec {
	return &histogramVec{name: name, help: help, label: label, series: map[string]*histogram{}}
}

func (h *histogramVec) Observe(v float64, labelValue string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	s, ok := h.series[labelValue]
	if !ok {
		s = &histogram{buckets: make([]uint64, len(defaultBuckets))}
		h.series[labelValue] = s
	}
	for i, le := range defaultBuckets {
		if v <= le {
			s.buckets[i]++
			break
		}
	}
	s.sum += v
	s.count++
}

func (h *histogramVec) write(w io.Writer) {
	h.mu.Lock()
	defer h.mu.Unlock()

	keys := make([]string, 0, len(h.series))
	for k := range h.series {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s histogram\n", h.name, h.help, h.name)
	for _, k := range keys {
		s := h.series[k]
		var cumulative uint64
		for i, le := range defaultBuckets {
			cumulative += s.buckets[i]
			fmt.Fprintf(w, "%s_bucket%s %d\n", h.name, formatLabels([]string{h.label, k, "le", formatValue(le)}), cumulative)
		}
		fmt.Fprintf(w, "%s_bucket%s %d\n", h.name, formatLabels([]string{h.label, k, "le", "+Inf"}), s.count)
		fmt.Fprintf(w, "%s_sum%s %s\n", h.name, formatLabels([]string{h.label, k}), formatValue(s.sum))
		fmt.Fprintf(w, "%s_count%s %d\n", h.name, formatLabels([]string{h.label, k}), s.count)
	}
}

func zipLabels(names, values []string) []string {
	pairs := make([]string, 0, len(names)*2)
	for i, name := range names {
		pairs = append(pairs, name, values[i])
	}
	return pairs
}

func sortSamples(samples []sample) {
	sort.Slice(samples, func(i, j int) bool {
		return strings.Join(samples[i].labels, "\xff") < strings.Join(samples[j].labels, "\xff")
	})
}

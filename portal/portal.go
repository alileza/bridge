package portal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"

	"github.com/alileza/bridge/audit"
	"github.com/alileza/bridge/auth"
	"github.com/alileza/bridge/httpredirector"
)

type Server struct {
	o          *Options
	srv        *http.Server
	metricsSrv *http.Server
}

type Options struct {
	ListenAddress string
	Logger        *log.Logger

	Redirector *httpredirector.HTTPRedirector

	UIProxyEnabled bool
	UIProxyURL     string

	// Version is reported in the bridge_build_info metric.
	Version string
	// MetricsEnabled exposes Prometheus metrics at /metrics.
	MetricsEnabled bool
	// MetricsAddress, when set, serves /metrics on this separate address
	// instead of the main listener (e.g. to keep it off the public port).
	MetricsAddress string

	// Auth, when set, requires GitHub login for the portal UI and API.
	// Short-link redirects stay public.
	Auth *auth.GitHub
	// Audit, when set, records who created, updated or deleted routes.
	Audit *audit.Log
}

// routeView is a route as returned by the API, with its last change when audited.
type routeView struct {
	Key       string     `json:"key"`
	URL       string     `json:"url"`
	UpdatedBy string     `json:"updated_by,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

func NewServer(o *Options) *Server {
	apiMux := http.NewServeMux()

	if o.Logger == nil {
		o.Logger = log.New(os.Stdout, "portal: ", log.LstdFlags)
	}

	metrics := NewMetrics(o.Version, o.Redirector.Storage)

	// protect requires a logged-in user for API handlers when auth is enabled.
	protect := func(h http.HandlerFunc) http.HandlerFunc {
		if o.Auth == nil {
			return h
		}
		return o.Auth.RequireAPI(h)
	}
	actor := func(r *http.Request) auth.Identity {
		if o.Auth != nil {
			if id, ok := o.Auth.User(r); ok {
				return id
			}
		}
		return auth.Identity{Login: "anonymous"}
	}
	record := func(e audit.Entry) {
		if o.Audit == nil {
			return
		}
		if err := o.Audit.Record(e); err != nil {
			o.Logger.Println("audit:", err)
			metrics.storageErrors.Inc("audit")
		}
	}

	withActor := func(id auth.Identity, e audit.Entry) audit.Entry {
		e.Actor, e.ActorEmails = id.Login, id.Emails
		return e
	}

	if o.Auth != nil {
		o.Auth.Register(apiMux)
	}

	apiMux.HandleFunc("GET /api/me", func(w http.ResponseWriter, r *http.Request) {
		me := map[string]any{"auth_enabled": o.Auth != nil, "authenticated": false}
		if o.Auth != nil {
			if id, ok := o.Auth.User(r); ok {
				me["authenticated"], me["login"], me["emails"] = true, id.Login, id.Emails
			}
		}
		responseOk(w, me)
	})

	apiMux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		keyWithHost := r.Host + r.URL.Path
		// lookup the key in the storage, if it exists, redirect
		dest, err := o.Redirector.Storage.Get(keyWithHost)
		if err == nil {
			metrics.redirects.Inc(r.Host)
			metrics.forwarded.Inc(keyWithHost)
			http.Redirect(w, r, dest, http.StatusFound)
			return
		} else {
			o.Logger.Printf("404 - GET %s ", r.URL.Path)
		}

		if o.Auth != nil && !o.Auth.RequirePage(w, r) {
			return
		}

		switch r.URL.Path {
		case "/":
			r.URL.Path = "/index.html"
		case "/favicon.ico":
			r.URL.Path = "/favicon.png"
		}

		b, err := assets.ReadFile("ui/dist" + r.URL.Path)
		if err != nil {
			log.Println("Error reading file", err.Error())
			metrics.misses.Inc(r.Host)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		if ct := mime.TypeByExtension(path.Ext(r.URL.Path)); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		w.Write(b)
	})

	apiMux.HandleFunc("GET /api/routes", protect(func(w http.ResponseWriter, r *http.Request) {
		o.Logger.Printf("200 - GET /api/routes\n")
		routes, err := o.Redirector.ListRoutes(r.Host)
		if err != nil {
			o.Logger.Println("500 - GET /api/routes: error listing routes:", err)
			responseError(w, err, http.StatusInternalServerError)
			return
		}
		views := make([]routeView, 0, len(routes))
		for _, route := range routes {
			v := routeView{Key: route.Key, URL: route.URL}
			if o.Audit != nil {
				if e, ok := o.Audit.Last(route.Key); ok {
					v.UpdatedBy, v.UpdatedAt = e.Actor, &e.Time
				}
			}
			views = append(views, v)
		}
		responseOk(w, views)
	}))

	apiMux.HandleFunc("GET /api/audit", protect(func(w http.ResponseWriter, r *http.Request) {
		if o.Audit == nil {
			responseOk(w, []audit.Entry{})
			return
		}
		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil || limit <= 0 || limit > 1000 {
			limit = 100
		}
		responseOk(w, o.Audit.Recent(limit, r.URL.Query().Get("key")))
	}))

	apiMux.HandleFunc("GET /api/routes/barcode", protect(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		url := r.Form.Get("url")
		if url == "" {
			o.Logger.Println("400 - GET /api/routes/preview: empty url")
			responseError(w, fmt.Errorf("empty url"), http.StatusBadRequest)
			return
		}

		o.Logger.Printf("0 - GET /api/routes/preview: %s\n", url)

		img, err := generateBarcode(url)
		if err != nil {
			o.Logger.Println("500 - GET /api/routes/preview: error generating barcode:", err)
			responseError(w, err, http.StatusInternalServerError)
			return
		}

		buffer := new(bytes.Buffer)
		if err := png.Encode(buffer, img); err != nil {
			o.Logger.Println("500 - GET /api/routes/preview: error encoding barcode:", err)
			responseError(w, err, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", buffer.Len()))
		w.WriteHeader(http.StatusOK)
		w.Write(buffer.Bytes())
	}))

	apiMux.HandleFunc("PUT /api/routes", protect(func(w http.ResponseWriter, r *http.Request) {
		var request httpredirector.Route
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			o.Logger.Println("400 - PUT /api/routes: error decoding request:", err)
			responseError(w, err, http.StatusBadRequest)
			return
		}

		if request.Key == "" {
			o.Logger.Println("400 - PUT /api/routes: empty key")
			responseError(w, fmt.Errorf("empty key"), http.StatusBadRequest)
			return
		}

		if request.URL == "" {
			o.Logger.Println("400 - PUT /api/routes: empty url")
			responseError(w, fmt.Errorf("empty url"), http.StatusBadRequest)
			return
		}

		o.Logger.Printf("202 - PUT /api/routes: %s -> %s\n", request.Key, request.URL)
		fullKey := httpredirector.RouteKey(r.Host, request.Key)
		previous, getErr := o.Redirector.Storage.Get(fullKey)
		if err := o.Redirector.SetRoute(r, request.Key, request.URL); err != nil {
			o.Logger.Println("400 - PUT /api/routes: error setting route:", err)
			metrics.storageErrors.Inc("set")
			responseError(w, err, http.StatusBadRequest)
			return
		}
		metrics.routeChanges.Inc("set")
		switch {
		case getErr != nil:
			record(withActor(actor(r), audit.Entry{Action: audit.ActionCreate, Key: fullKey, URL: request.URL}))
		case previous != request.URL:
			record(withActor(actor(r), audit.Entry{Action: audit.ActionUpdate, Key: fullKey, URL: request.URL, PreviousURL: previous}))
		}
		w.WriteHeader(http.StatusAccepted)
	}))

	apiMux.HandleFunc("DELETE /api/routes", protect(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Key string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			o.Logger.Println("400 - DELETE /api/routes: error decoding request:", err)
			responseError(w, err, http.StatusBadRequest)
			return
		}
		o.Logger.Printf("200 - DELETE /api/routes: %s\n", request.Key)
		previous, _ := o.Redirector.Storage.Get(request.Key)
		if err := o.Redirector.RemoveRoute(request.Key); err == nil {
			metrics.routeChanges.Inc("delete")
			record(withActor(actor(r), audit.Entry{Action: audit.ActionDelete, Key: request.Key, PreviousURL: previous}))
		}
		w.WriteHeader(http.StatusOK)
	}))

	apiMux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if _, err := o.Redirector.Storage.List(); err != nil {
			metrics.storageErrors.Inc("list")
			responseError(w, err, http.StatusServiceUnavailable)
			return
		}
		responseOk(w, map[string]string{"status": "ok"})
	})

	s := &Server{
		o: o,
		srv: &http.Server{
			Addr:    o.ListenAddress,
			Handler: metrics.Instrument(apiMux),
		},
	}

	switch {
	case o.MetricsAddress != "":
		metricsMux := http.NewServeMux()
		metricsMux.Handle("GET /metrics", metrics)
		s.metricsSrv = &http.Server{Addr: o.MetricsAddress, Handler: metricsMux}
	case o.MetricsEnabled:
		apiMux.Handle("GET /metrics", metrics)
	}

	return s
}

func responseOk(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func responseError(w http.ResponseWriter, err error, code int) {
	if err == nil {
		err = fmt.Errorf("%d", code)
	}
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}

func (s *Server) Start() error {
	errc := make(chan error, 2)
	if s.metricsSrv != nil {
		s.o.Logger.Printf("Serving metrics on %s/metrics\n", s.o.MetricsAddress)
		go func() { errc <- s.metricsSrv.ListenAndServe() }()
	}
	s.o.Logger.Printf("Listening on %s\n", s.o.ListenAddress)
	go func() { errc <- s.srv.ListenAndServe() }()
	return <-errc
}

func generateBarcode(url string) (image.Image, error) {
	qrCode, err := qr.Encode(url, qr.L, qr.Auto)
	if err != nil {
		return nil, err
	}
	qrCode, err = barcode.Scale(qrCode, 200, 200)
	if err != nil {
		return nil, err
	}
	return qrCode, nil
}

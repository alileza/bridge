package portal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/alileza/bridge/httpredirector"
)

var (
	forwardCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bridge_routes_forwarded_total",
			Help: "Total number of requests to the forward handler.",
		},
		[]string{"key"},
	)
	routesRegistered = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "bridge_routes_registered_count",
			Help: "Total number of requests to the forward handler.",
		},
		[]string{"host"},
	)
)

func init() {
	prometheus.MustRegister(forwardCounter)
}

type Server struct {
	o   *Options
	srv *http.Server
}

type Options struct {
	ListenAddress string
	Logger        *log.Logger

	Redirector *httpredirector.HTTPRedirector

	UIProxyEnabled bool
	UIProxyURL     string
}

func NewServer(o *Options) *Server {
	apiMux := http.NewServeMux()

	if o.Logger == nil {
		o.Logger = log.New(os.Stdout, "portal: ", log.LstdFlags)
	}

	apiMux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		keyWithHost := r.Host + r.URL.Path
		// lookup the key in the storage, if it exists, redirect
		dest, err := o.Redirector.Storage.Get(keyWithHost)
		if err == nil {
			forwardCounter.With(prometheus.Labels{"key": keyWithHost}).Inc()
			http.Redirect(w, r, dest, http.StatusFound)
			return
		} else {
			o.Logger.Printf("404 - GET %s ", r.URL.Path)
		}

		if r.URL.Path == "/" {
			r.URL.Path = "/index.html"
		}

		b, err := assets.ReadFile("ui/dist" + r.URL.Path)
		if err != nil {
			log.Println("Error reading file", err.Error())
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		}
		if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		}
		w.Write(b)
	})

	apiMux.HandleFunc("GET /api/routes", func(w http.ResponseWriter, r *http.Request) {
		o.Logger.Printf("200 - GET /api/routes\n")
		routes, err := o.Redirector.ListRoutes(r.Host)
		if err != nil {
			o.Logger.Println("500 - GET /api/routes: error listing routes:", err)
			responseError(w, err, http.StatusInternalServerError)
			return
		}
		responseOk(w, routes)
	})

	apiMux.HandleFunc("GET /api/routes/barcode", func(w http.ResponseWriter, r *http.Request) {
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
	})

	apiMux.HandleFunc("PUT /api/routes", func(w http.ResponseWriter, r *http.Request) {
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
		if err := o.Redirector.SetRoute(r, request.Key, request.URL); err != nil {
			o.Logger.Println("400 - PUT /api/routes: error setting route:", err)
			responseError(w, err, http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})

	apiMux.HandleFunc("DELETE /api/routes", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Key string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			o.Logger.Println("400 - DELETE /api/routes: error decoding request:", err)
			responseError(w, err, http.StatusBadRequest)
			return
		}
		o.Logger.Printf("200 - DELETE /api/routes: %s\n", request.Key)
		o.Redirector.RemoveRoute(request.Key)
		w.WriteHeader(http.StatusOK)
	})

	apiMux.HandleFunc("GET /metrics", promhttp.Handler().ServeHTTP)

	srv := &http.Server{
		Addr:    o.ListenAddress,
		Handler: apiMux,
	}

	return &Server{
		o:   o,
		srv: srv,
	}
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
	s.o.Logger.Printf("Listening on %s\n", s.o.ListenAddress)
	return s.srv.ListenAndServe()
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

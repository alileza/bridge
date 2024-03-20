package portal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"log"
	"net/http"
	"os"

	"bridge/httpredirector"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	forwardCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_forward_requests_total",
			Help: "Total number of requests to the forward handler.",
		},
		[]string{"key"},
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

	uiHandler := NewUIHandler(o.UIProxyURL)

	apiMux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		keyWithHost := r.Host + r.URL.Path
		dest, ok := o.Redirector.Storage.Load(keyWithHost)
		if ok {
			forwardCounter.With(prometheus.Labels{"key": keyWithHost}).Inc()
			http.Redirect(w, r, dest.(string), http.StatusFound)
			return
		}

		// o.Logger.Printf("GET %s ", r.URL.Path)
		if o.UIProxyEnabled {
			uiHandler.ProxyHandler(w, r)
		} else {
			uiHandler.StaticFileHandler(w, r)
		}
	})

	apiMux.HandleFunc("GET /api/routes", func(w http.ResponseWriter, r *http.Request) {
		o.Logger.Printf("200 - GET /api/routes\n")
		responseOk(w, o.Redirector.ListRoutes(r.Host))
	})

	apiMux.HandleFunc("GET /api/routes/preview", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		url := r.Form.Get("url")

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

func encodeImageToBase64(img image.Image) string {
	// Encode the image as PNG
	buf := new(bytes.Buffer)
	_ = png.Encode(buf, img)

	// Encode the PNG image as base64
	encodedStr := base64.StdEncoding.EncodeToString(buf.Bytes())

	return encodedStr
}

package portal

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"bridge/httpredirector"
	"bridge/opengraph"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	o   *Options
	srv *http.Server
}

type Options struct {
	ListenAddress string
	Logger        *log.Logger

	Redirector *httpredirector.HTTPRedirector

	UIStaticFilepath string
	UIProxyEnabled   bool
	UIProxyURL       string
}

func NewServer(o *Options) *Server {
	apiMux := http.NewServeMux()

	if o.Logger == nil {
		o.Logger = log.New(os.Stdout, "portal: ", log.LstdFlags)
	}

	uiHandler := NewUIHandler(o.UIStaticFilepath, o.UIProxyURL)

	apiMux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if url := o.Redirector.GetRedirectURL(r.URL); url == "https://alileza.me/" {
			o.Logger.Printf("GET %s ", r.URL.Path)
			if o.UIProxyEnabled {
				uiHandler.ProxyHandler(w, r)
			} else {
				uiHandler.StaticFileHandler(w, r)
			}
			return
		}

		o.Redirector.Handler(w, r)
	})

	apiMux.HandleFunc("GET /api/routes", func(w http.ResponseWriter, r *http.Request) {
		o.Logger.Printf("200 - GET /api/routes\n")
		o.Redirector.BaseURL = "https://" + r.Host
		responseOk(w, o.Redirector.ListRoutes())
	})

	apiMux.HandleFunc("GET /api/routes/preview", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		url := r.Form.Get("url")

		o.Logger.Printf("0 - GET /api/routes/preview: %s\n", url)

		img, err := opengraph.GenerateBarcode(url)
		if err != nil {
			o.Logger.Println("500 - GET /api/routes/preview: error generating barcode:", err)
			responseError(w, err, http.StatusInternalServerError)
			return
		}
		b64image := opengraph.EncodeImageToBase64(img)

		responseOk(w, map[string]any{
			"image": b64image,
		})
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
		if err := o.Redirector.SetRoute(request.Key, request.URL); err != nil {
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

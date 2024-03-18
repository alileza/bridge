package portal

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"bridge/httpredirector"
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
		o.Logger.Printf("GET /api/routes\n")
		responseOk(w, o.Redirector.ListRoutes())
	})

	apiMux.HandleFunc("PUT /api/routes", func(w http.ResponseWriter, r *http.Request) {
		var request httpredirector.Route
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			o.Logger.Println("PUT /api/routes: error decoding request:", err)
			responseError(w, err, http.StatusBadRequest)
			return
		}

		o.Logger.Printf("PUT /api/routes: %s -> %s\n", request.Key, request.URL)
		o.Redirector.SetRoute(request.Key, request.URL)
	})

	apiMux.HandleFunc("DELETE /api/routes", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Key string `json:"key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			o.Logger.Println("DELETE /api/routes: error decoding request:", err)
			responseError(w, err, http.StatusBadRequest)
			return
		}
		o.Logger.Printf("DELETE /api/routes: %s\n", request.Key)
		o.Redirector.RemoveRoute(request.Key)
	})

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
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}

func (s *Server) Start() error {
	s.o.Logger.Printf("Listening on %s\n", s.o.ListenAddress)
	return s.srv.ListenAndServe()
}

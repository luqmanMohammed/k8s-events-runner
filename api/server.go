package api

import (
	"encoding/json"
	"net/http"
)

var (
	healthResponse = map[string]string{
		"message": "healthy",
	}
)

type erServer struct {
	host           string `deafult:""`
	port           int    `default:"8080"`
	caCertPath     string
	serverKeyPath  string
	serverCertPath string
	serveMux       *http.ServeMux
}

func New(host string, port int, caCertPath, serverKeyPath string, serverCertPath string) *erServer {
	erSer := &erServer{
		host:           host,
		port:           port,
		caCertPath:     caCertPath,
		serverKeyPath:  serverCertPath,
		serverCertPath: serverCertPath,
		serveMux:       http.DefaultServeMux,
	}
	erSer.registerRoutes()
	return erSer
}

func (ers erServer) ListenNoTLS() error {
	
	server := &http.Server{
		Addr:    ":8080",
		Handler: ers.serveMux,
	}
	return server.ListenAndServe()
}

func (ers *erServer) registerRoutes() {
	ers.serveMux.HandleFunc("/api/v1/health", healthHandler)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthResponse)
}

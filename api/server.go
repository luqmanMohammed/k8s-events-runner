package api

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"k8s.io/klog/v2"
)

var (
	healthResponse = map[string]string{
		"message": "healthy",
	}
)

type erServer struct {
	addr     string `default:":8080"`
	serveMux *http.ServeMux
}

func New(addr string) *erServer {
	erSer := &erServer{
		addr:     addr,
		serveMux: http.DefaultServeMux,
	}
	erSer.registerRoutes()
	return erSer
}

func (ers erServer) ListenNoTLS() error {
	klog.Infof("Server listening on %s", ers.addr)
	server := &http.Server{
		Addr:    ers.addr,
		Handler: ers.serveMux,
	}
	return server.ListenAndServe()
}

func (ers erServer) ListenMTLS(caCertPath, serverKeyPath, serverCertPath string) error {
	caCert, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		return err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}

	klog.Infof("MTLS Server listening on %s", ers.addr)
	klog.Info("Using MTLS based authentication")
	if klog.V(2).Enabled() {
		klog.Info("Use Server's CA Key to sign Client Cert")
		klog.Info("Server's CA Key: ", serverKeyPath)
	}
	server := &http.Server{
		Addr:      ers.addr,
		TLSConfig: tlsConfig,
	}

	return server.ListenAndServeTLS(serverCertPath, serverKeyPath)
}
func (ers *erServer) registerRoutes() {
	ers.serveMux.HandleFunc("/api/v1/health", healthHandler)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthResponse)
}

package api

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/luqmanMohammed/k8s-events-runner/config"
	queue "github.com/luqmanMohammed/k8s-events-runner/queue"
	"k8s.io/klog/v2"
)

var (
	//static health response
	healthResponse = baseResponse{
		Message: "OK",
	}
)

//baseResponse is a generic response struct with a message field
type baseResponse struct {
	Message string `json:"message"`
}

//event is used to parse the request body which ideally should be a json respresentation of a k8s event
type event struct {
	EventType    string                 `json:"type"`
	ResourseType string                 `json:"resourseType"`
	Object       map[string]interface{} `json:"object"`
}

type erServer struct {
	addr            string `default:":8080"`
	serveMux        *http.ServeMux
	jobQueue        *queue.JobQueue
	configCollector config.ConfigCollector
}

func New(addr string, jq *queue.JobQueue, cc config.ConfigCollector) *erServer {
	erSer := &erServer{
		addr:            addr,
		jobQueue:        jq,
		configCollector: cc,
		serveMux:        http.DefaultServeMux,
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
	ers.serveMux.HandleFunc("/api/v1/event", ers.eventHandler)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(healthResponse)
}

func (ers *erServer) eventHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	} else {
		var event event
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(baseResponse{Message: "Invalid or No Request Body"})
			return
		}
		err = json.Unmarshal(body, &event)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(baseResponse{Message: "Invalid Request Body"})
			return
		}
		klog.V(1).Info("Received event", "event", event)
		rva, err := ers.configCollector.GetRunnerConfigForResourceAndEvent(event.ResourseType, event.EventType)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(baseResponse{Message: fmt.Sprintf("No Runner Config Found for %s:%s", event.ResourseType, event.EventType)})
			return
		}
		job := queue.Job{
			RunnerConfig: rva,
			EventType:    event.EventType,
			Resource:     event.ResourseType,
		}
		ers.jobQueue.AddJob(&job)
		w.WriteHeader(http.StatusCreated)
	}
}

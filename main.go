package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (c *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	metricsInc := func(resp http.ResponseWriter, req *http.Request) {
		c.fileserverHits.Add(1)
		next.ServeHTTP(resp, req)
	}
	return http.HandlerFunc(metricsInc)
}

func (c *apiConfig) handleHitResponse(resp http.ResponseWriter, req *http.Request) {
	hits := c.fileserverHits.Load()
	resp.WriteHeader(http.StatusOK)
	fmt.Fprintf(resp, "Hits: %d", hits)
}

func (c *apiConfig) handleReset(resp http.ResponseWriter, req *http.Request) {
	c.fileserverHits.Store(0)
	resp.WriteHeader(http.StatusOK)
	fmt.Fprint(resp, "Hits reset to 0")
}

func healthStatus(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Add("Content-Type", "text/plain; charset=utf-8")
	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte("OK"))
}

func main() {
	serveMux := http.NewServeMux()
	server := http.Server{
		Handler: serveMux,
		Addr:    ":8080",
	}

	conf := apiConfig{}

	appHandler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))
	serveMux.Handle("/app/", conf.middlewareMetricsInc(appHandler))
	serveMux.HandleFunc("GET /api/healthz", healthStatus)
	serveMux.HandleFunc("GET /api/metrics", conf.handleHitResponse)
	serveMux.HandleFunc("POST /api/reset", conf.handleReset)

	log.Println("Starting HTTP server on port 8080")
	log.Fatal(server.ListenAndServe())
}

package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	FServerHits atomic.Int32
}


func (config *apiConfig) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config.FServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	apiCfg := apiConfig{}
	mux := http.NewServeMux()

	// main entry
	fileServerAppHandler := http.StripPrefix("/app/", http.FileServer(http.Dir("./app/")))
	mux.Handle("/app/", apiCfg.metricsMiddleware(fileServerAppHandler))
	
	// assets
	fileServerAssetsHandler := http.StripPrefix("/app/assets", http.FileServer(http.Dir("./app/assets/")))
	mux.Handle("/app/assets/", fileServerAssetsHandler)

	// server health
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// metrics - hits
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		hits := fmt.Sprintf("Hits: %d", apiCfg.FServerHits.Load())
		
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(hits))
	})

	// metrics - reset
	mux.HandleFunc("POST /reset", func(w http.ResponseWriter, r *http.Request) {
		apiCfg.FServerHits.Store(0)
		w.WriteHeader(http.StatusOK)
	})
	

	server := &http.Server{
		Addr: ":" + "8080",
		Handler: mux,
	}

	log.Printf("Serving on port: 8080")
	log.Fatal(server.ListenAndServe())
}

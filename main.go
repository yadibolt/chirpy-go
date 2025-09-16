package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./")))

	server := &http.Server{
		Addr: ":" + "8080",
		Handler: mux,
	}

	log.Printf("Serving on port: 8080")
	log.Fatal(server.ListenAndServe())
}

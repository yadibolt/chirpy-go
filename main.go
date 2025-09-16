package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr: ":" + "8080",
		Handler: mux,
	}

	log.Printf("Serving on port: 8080")
	log.Fatal(server.ListenAndServe())
}

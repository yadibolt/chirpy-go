package main
import _ "github.com/lib/pq"
import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"encoding/json"
	"strings"
	"slices"
)

type apiConfig struct {
	FServerHits atomic.Int32
}

type RequestJsonStruct struct {
	Body string `json:"body"`
}

type ErrorJsonResponse struct {
	Error string `json:"error"`
}

type ValidJsonResponse struct {
	Valid bool `json:"valid"`
}

type CleanedJsonResponse struct {
	CleanedBody string `json:"cleaned_body"`
}

func (config *apiConfig) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config.FServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	pload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error parsing JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write([]byte(pload))
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	mes := ErrorJsonResponse{
		Error: message,
	}

	m, err := json.Marshal(mes)
	if err != nil {
		log.Printf("Error parsing JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write([]byte(m))
}

func checkProfane(message string) string {
	bannedWords := []string{"kerfuffle", "sharbert", "fornax"}
	messageSlice := strings.Split(message, " ")
	for index, value := range messageSlice {
		messageToCheck := strings.ToLower(value)
		if slices.Contains(bannedWords, messageToCheck) {
			messageSlice[index] = "****"
		}
	}

	finalMessage := strings.Join(messageSlice, " ")

	return finalMessage
}

func main() {
	apiCfg := apiConfig{}
	mux := http.NewServeMux()

	// main entry
	fileServerAppHandler := http.StripPrefix("/app/", http.FileServer(http.Dir("./app/")))
	mux.Handle("/app/", apiCfg.metricsMiddleware(fileServerAppHandler))
	
	// assets
	fileServerAssetsHandler := http.StripPrefix("/app/assets", http.FileServer(http.Dir("./app/assets/")))
	mux.Handle("/app/assets/", apiCfg.metricsMiddleware(fileServerAssetsHandler))

	// server health
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// metrics - hits
	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		hits := apiCfg.FServerHits.Load()
		template := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", hits)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(template))
	})

	// metrics - reset
	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		apiCfg.FServerHits.Store(0)
		w.WriteHeader(http.StatusOK)
	})

	// chirp validator
	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		dec := json.NewDecoder(r.Body)
		params := RequestJsonStruct{}
		err := dec.Decode(&params)

		if err != nil {		
			respondWithError(w, 500, "Something went wrong")
			return
		}

		if len(params.Body) > 140 {
			respondWithError(w, 400, "Chirp is too long")
			return
		}

		message := checkProfane(params.Body)
		mes := CleanedJsonResponse{
			CleanedBody: message,
		}
		respondWithJson(w, 200, mes)
	})

	server := &http.Server{
		Addr: ":" + "8080",
		Handler: mux,
	}

	log.Printf("Serving on port: 8080")
	log.Fatal(server.ListenAndServe())
}

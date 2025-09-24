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
	"os"
	"database/sql"
	"github.com/yadibolt/chirpy-go/internal/database"
	"github.com/joho/godotenv"
	"time"
	"github.com/google/uuid"
)

type apiConfig struct {
	FServerHits atomic.Int32
	DBQueries *database.Queries 
	IsDev string
}

type RequestJsonStruct struct {
	Body string `json:"body"`
}

type RequestChirpsStruct struct {
	Body string `json:"body"`
	UserId uuid.UUID `json:"user_id"`
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

type PostUsersJsonRequest struct {
	Email string `json:"email"`
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID	uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body	string	`json:"body"`
	UserId	uuid.UUID `json:"user_id"`
}

type CreateChirpParams struct {
	Body string
	UserId	uuid.UUID
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
	godotenv.Load()

	databaseURL := os.Getenv("DB_URL")
	isDev := os.Getenv("PLATFORM")
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Printf("Could not connect to the database.")
		return
	} else {
		log.Printf("Connection to the database established.")
	}

	apiCfg := apiConfig{}
	dbQueries := database.New(db)
	apiCfg.DBQueries = dbQueries
	apiCfg.IsDev = isDev
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

	// metrics - hit
	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		hits := apiCfg.FServerHits.Load()
		template := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", hits)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		w.Write([]byte(template))
	})

	// metrics - reset
	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {

		if isDev == "dev" {
			dbQueries.DeleteChirps(r.Context())
			dbQueries.DeleteUsers(r.Context())
		} else {
			w.WriteHeader(403)
		}

		apiCfg.FServerHits.Store(0)
		w.WriteHeader(http.StatusOK)
	})

	// chirp validator
	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		dec := json.NewDecoder(r.Body)
		params := RequestChirpsStruct{}
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
		newChirp := database.CreateChirpParams{
			Body: message,
			UserID: params.UserId,
		}

		dbChirp, err := dbQueries.CreateChirp(r.Context(), newChirp)
		if err != nil {
			respondWithError(w, 500, "Could not create chirp")
			return
		}

		chirpJson := Chirp{
			ID: dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body: dbChirp.Body,
			UserId: dbChirp.UserID,
		}

		respondWithJson(w, 201, chirpJson)
	})

	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {	
		allChirps := []Chirp{}
		dbChirps, err := dbQueries.GetChirps(r.Context())
		if err != nil {
			respondWithError(w, 500, "could not retrieve")
			return
		}

		if l:=len(dbChirps); l <= 0 {
			respondWithError(w, 500, "could not check size")
			return
		}

		for i := 0; i < len(dbChirps); i++ {
			newChirpJson := Chirp{
				ID: dbChirps[i].ID,
				CreatedAt: dbChirps[i].CreatedAt,
				UpdatedAt: dbChirps[i].UpdatedAt,
				Body: dbChirps[i].Body,
				UserId: dbChirps[i].UserID,
			}

			allChirps = append(allChirps, newChirpJson)
		}

		respondWithJson(w, 200, allChirps)
	})

	mux.HandleFunc("GET /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {
		pathVal := r.PathValue("chirpID")

		if len(pathVal) <= 0 {
			respondWithError(w, 400, "No chirp specified")
			return
		}

		fmt.Printf("Pathval %s", pathVal)

		uuidFromString, err := uuid.Parse(pathVal);
		if err != nil {
			respondWithError(w, 400, "Could not parse id")
			return
		}

		dbChirp, err := dbQueries.GetChirp(r.Context(), uuidFromString)
		if err != nil {
			respondWithError(w, 404, "Did not find a chirp")
			return
		}

		chirp := Chirp{
			ID: dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body: dbChirp.Body,
			UserId: dbChirp.UserID,
		}

		respondWithJson(w, 200, chirp)
	})

	// create user
	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request){
		dec := json.NewDecoder(r.Body)
		params := PostUsersJsonRequest{}
		err := dec.Decode(&params)
		if err != nil {
			respondWithError(w, 500, "Something went wrong: Could not decode params.")
			return
		}

		user, err := dbQueries.CreateUser(r.Context(), params.Email)
		if err != nil {
			mess := fmt.Sprintf("Something went wrong: %s", err)
			respondWithError(w, 500, mess)
			return
		}


		apiUser := User{
			ID: user.ID,
			CreatedAt:user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email: user.Email,
		}

		respondWithJson(w, 201, apiUser)
	})

	server := &http.Server{
		Addr: ":" + "8080",
		Handler: mux,
	}

	log.Printf("Serving on port: 8080")
	log.Fatal(server.ListenAndServe())
}

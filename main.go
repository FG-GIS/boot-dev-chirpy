package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/FG-GIS/boot-dev-chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type validChirp struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	CleansedBody string    `json:"body"`
	UserID       uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
func (cfg *apiConfig) metricsEnd(w http.ResponseWriter, r *http.Request) {
	res := []byte{}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(fmt.Appendf(res, `
	<html>
		<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body>
	</html>
		`, cfg.fileserverHits.Load()))
}
func (cfg *apiConfig) metricsReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(w, 403, "Endpoint limited for development access.")
	}
	cfg.dbQueries.Reset(r.Context())
	res := []byte{}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(fmt.Appendf(res, "Hits reset from: %v\nTo: 0\nUsers table reset.", cfg.fileserverHits.Swap(0)))
}

func respondWithError(w http.ResponseWriter, code int, errorMsg string) {
	log.Print(errorMsg)
	w.WriteHeader(code)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error marshaling JSON: %s", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func profaneCensor(msg string) string {
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	msgSlice := strings.Split(msg, " ")
	for idx, word := range msgSlice {
		if slices.Contains(badWords, strings.ToLower(word)) {
			msgSlice[idx] = "****"
		}
	}
	return strings.Join(msgSlice, " ")
}

func (cfg *apiConfig) validationHandler(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	message := chirp{}
	err := decoder.Decode(&message)
	code := 200
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error decoding the message: %s", err))
	}
	if len([]rune(message.Body)) > 140 {
		code = 400
		respondWithError(w, code, "Chirp is too long.")
		return
	}
	msg := profaneCensor(message.Body)
	usr, err := cfg.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   msg,
		UserID: message.UserID,
	})
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error creating chirp record: %s", err))
	}
	respBody := validChirp{
		ID:           usr.ID,
		CreatedAt:    usr.CreatedAt,
		UpdatedAt:    usr.UpdatedAt,
		CleansedBody: usr.Body,
		UserID:       usr.UserID,
	}
	code = 201

	respondWithJSON(w, code, respBody)
}

func (cfg *apiConfig) addUser(w http.ResponseWriter, r *http.Request) {
	type userData struct {
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	mail := userData{}
	err := decoder.Decode(&mail)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error decoding message: %s", err))
	}
	dbUser, err := cfg.dbQueries.CreateUser(r.Context(), mail.Email)
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Error creating user: %s", err))
	}
	user := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}
	respondWithJSON(w, 201, user)
}

func (cfg *apiConfig) getChirps(w http.ResponseWriter, r *http.Request) {
	rawChirpSlice, err := cfg.dbQueries.GetChirps(r.Context())
	chirps := []validChirp{}
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error retrieving chirps from database: %v", err))
	}
	for _, chi := range rawChirpSlice {
		chirps = append(chirps, validChirp{
			ID:           chi.ID,
			CreatedAt:    chi.CreatedAt,
			UpdatedAt:    chi.UpdatedAt,
			CleansedBody: chi.Body,
			UserID:       chi.UserID,
		})
	}
	respondWithJSON(w, 200, chirps)
}

func (cfg *apiConfig) getChirpById(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Error converting chirp ID: %s", err))
	}
	chirp, err := cfg.dbQueries.GetChirpByID(r.Context(), id)
	if err != nil {
		respondWithError(w, 404, fmt.Sprintf("Error chirp not found: %s", err))
	}
	vChirp := validChirp{
		ID:           chirp.ID,
		CreatedAt:    chirp.CreatedAt,
		UpdatedAt:    chirp.UpdatedAt,
		CleansedBody: chirp.Body,
		UserID:       chirp.UserID,
	}
	respondWithJSON(w, 200, vChirp)
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening database: %s", err)
	}

	apiCfg := apiConfig{
		dbQueries: database.New(db),
		platform:  os.Getenv("PLATFORM"),
	}
	port := "8080"
	filepathRoot := "/app/"
	apiPath := "/api"
	adminPath := "/admin"
	mux := http.NewServeMux()
	mux.Handle(filepathRoot, apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))

	mux.HandleFunc("GET "+apiPath+"/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET "+adminPath+"/metrics", apiCfg.metricsEnd)
	mux.HandleFunc("POST "+adminPath+"/reset", apiCfg.metricsReset)
	mux.HandleFunc("POST "+apiPath+"/chirps", apiCfg.validationHandler)
	mux.HandleFunc("GET "+apiPath+"/chirps", apiCfg.getChirps)
	mux.HandleFunc("POST "+apiPath+"/users", apiCfg.addUser)
	mux.HandleFunc("GET "+apiPath+"/chirps/{chirpID}", apiCfg.getChirpById)

	server := &http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}
	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}

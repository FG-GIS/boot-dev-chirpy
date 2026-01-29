package main

import (
	"context"
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

	"github.com/FG-GIS/boot-dev-chirpy/internal/auth"
	"github.com/FG-GIS/boot-dev-chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
	tknSecret      string
}

type User struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

type validChirp struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	CleansedBody string    `json:"body"`
	UserID       uuid.UUID `json:"user_id"`
}

type userData struct {
	Password string `json:"password"`
	Email    string `json:"email"`
	// ExpirationTime int    `json:"expires_in_seconds"`
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
		return
	}
	cfg.dbQueries.Reset(r.Context())
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(fmt.Appendf([]byte{}, "Hits reset from: %v\nTo: 0\nUsers table reset.", cfg.fileserverHits.Swap(0)))
}

func respondWithError(w http.ResponseWriter, code int, errorMsg string) {
	log.Print(errorMsg)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	_, err := w.Write(fmt.Append([]byte{}, errorMsg))
	if err != nil {
		log.Printf("Error w.Write: %s\n", err)
	}
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error marshaling JSON: %s", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(data)
	if err != nil {
		log.Printf("Error w.Write: %s\n", err)
	}
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

func (cfg *apiConfig) validateRefreshToken(h http.Header, cont context.Context) (database.RefreshToken, error) {
	rfrTk, err := auth.GetBearerToken(h)
	if err != nil {
		return database.RefreshToken{}, err
	}
	dbRfrTk, err := cfg.dbQueries.GetRefreshToken(cont, rfrTk)
	if err != nil {
		return database.RefreshToken{}, err
	}
	if !dbRfrTk.RevokedAt.Time.IsZero() {
		return database.RefreshToken{}, fmt.Errorf("Refresh token expired")
	}
	return dbRfrTk, nil
}

func (cfg *apiConfig) validateAccessToken(h http.Header) (uuid.UUID, error) {
	brToken, err := auth.GetBearerToken(h)
	if err != nil {
		return uuid.Nil, fmt.Errorf("Error retrieving authorization: %s", err)
	}
	userID, err := auth.ValidateJWT(brToken, cfg.tknSecret)
	if err != nil {
		return uuid.Nil, fmt.Errorf("Error unauthorized: %s", err)
	}
	return userID, nil
}

func (cfg *apiConfig) validationHandler(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}

	userID, err := cfg.validateAccessToken(r.Header)
	if err != nil {
		respondWithError(w, 401, fmt.Sprintf("Error validating acces token: %s", err))
		return
	}

	decoder := json.NewDecoder(r.Body)
	message := chirp{}
	err = decoder.Decode(&message)
	code := 200
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error decoding the message: %s", err))
		return
	}
	if len([]rune(message.Body)) > 140 {
		code = 400
		respondWithError(w, code, "Chirp is too long.")
		return
	}
	msg := profaneCensor(message.Body)
	usr, err := cfg.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   msg,
		UserID: userID,
	})
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error creating chirp record: %s", err))
		return
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
	decoder := json.NewDecoder(r.Body)
	usrData := userData{}
	err := decoder.Decode(&usrData)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error decoding message: %s", err))
		return
	}

	hashP, err := auth.HashPassword(usrData.Password)
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Error hashing password: %s", err))
		return
	}

	dbUser, err := cfg.dbQueries.CreateUser(r.Context(), database.CreateUserParams{
		Email:          usrData.Email,
		HashedPassword: hashP,
	})
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Error creating user: %s", err))
		return
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
		return
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
		respondWithError(w, 500, fmt.Sprintf("Error converting chirp ID: %s", err))
		return
	}
	chirp, err := cfg.dbQueries.GetChirpByID(r.Context(), id)
	if err != nil {
		respondWithError(w, 404, fmt.Sprintf("Error chirp not found: %s", err))
		return
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

func (cfg *apiConfig) userLogin(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	usrData := userData{}
	err := decoder.Decode(&usrData)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error decoding message: %s", err))
		return
	}
	usr, err := cfg.dbQueries.GetUserByMail(r.Context(), usrData.Email)
	if err != nil {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}
	check, err := auth.CheckPasswordHash(usrData.Password, usr.HashedPassword)
	if err != nil {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}
	if !check {
		respondWithError(w, 401, "Incorrect email or password")
		return
	}
	rfrToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Server error, creating refresh token: %s", err))
		return
	}
	rfrTokenEntry, err := cfg.dbQueries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:  rfrToken,
		UserID: usr.ID,
	})
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Server error, creating refresh token: %s", err))
		return
	}
	tkn, err := auth.MakeJWT(usr.ID, cfg.tknSecret, 1*time.Hour)
	usrResponse := User{
		ID:           usr.ID,
		CreatedAt:    usr.CreatedAt,
		UpdatedAt:    usr.UpdatedAt,
		Email:        usr.Email,
		Token:        tkn,
		RefreshToken: rfrTokenEntry.Token,
	}
	respondWithJSON(w, 200, usrResponse)
}

func (cfg *apiConfig) TkHandlerRefresh(w http.ResponseWriter, r *http.Request) {
	type token struct {
		Tk string `json:"token"`
	}
	dbRfrTk, err := cfg.validateRefreshToken(r.Header, r.Context())
	if err != nil {
		respondWithError(w, 401, fmt.Sprintf("Error with the refresh token: %s", err))
		return
	}
	dbUsr, err := cfg.dbQueries.GetUserFromRefreshToken(r.Context(), dbRfrTk.Token)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Database error: %s", err))
		return
	}
	respToken, err := auth.MakeJWT(dbUsr.ID, cfg.tknSecret, 1*time.Hour)
	respondWithJSON(w, 200, token{Tk: respToken})
}

func (cfg *apiConfig) revokeRefreshToken(w http.ResponseWriter, r *http.Request) {
	dbRfrTk, err := cfg.validateRefreshToken(r.Header, r.Context())
	if err != nil {
		respondWithError(w, 401, fmt.Sprintf("Error with the refresh token: %s", err))
		return
	}
	revokedTk, err := cfg.dbQueries.RevokeRefreshToken(r.Context(), database.RevokeRefreshTokenParams{
		Token: dbRfrTk.Token,
		RevokedAt: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
	})
	fmt.Printf("Token %s\nRevoked at:%s\n", revokedTk.Token, revokedTk.RevokedAt.Time.String())
	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) updateUser(w http.ResponseWriter, r *http.Request) {
	userID, err := cfg.validateAccessToken(r.Header)
	if err != nil {
		respondWithError(w, 401, fmt.Sprintf("Error validating access token: %s", err))
		return
	}
	usrData := userData{}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&usrData)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error decoding request: %s", err))
		return
	}
	newHash, err := auth.HashPassword(usrData.Password)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Server error: %s", err))
		return
	}
	usr, err := cfg.dbQueries.UpdateCredentials(r.Context(), database.UpdateCredentialsParams{
		ID:             userID,
		Email:          usrData.Email,
		HashedPassword: newHash,
	})

	usrResponse := User{
		ID:        usr.ID,
		CreatedAt: usr.CreatedAt,
		UpdatedAt: usr.UpdatedAt,
		Email:     usr.Email,
	}
	respondWithJSON(w, 200, usrResponse)
}

func (cfg *apiConfig) delChirpById(w http.ResponseWriter, r *http.Request) {
	tk, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, fmt.Sprintf("Unauthorized: %s", err))
		return
	}
	usrId, err := auth.ValidateJWT(tk, cfg.tknSecret)
	if err != nil {
		respondWithError(w, 403, fmt.Sprintf("Forbidden: %s", err))
		return
	}
	id, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error converting chirp ID: %s", err))
		return
	}
	chirp, err := cfg.dbQueries.GetChirpByID(r.Context(), id)
	if err != nil {
		respondWithError(w, 404, fmt.Sprintf("Error chirp not found: %s", err))
		return
	}
	if chirp.UserID != usrId {
		respondWithError(w, 403, "Forbidden")
		return
	}
	err = cfg.dbQueries.DeleteChirpByID(r.Context(), id)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Something went wrong: %s", err))
	}
	w.WriteHeader(204) // http.StatusNoContent
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
		tknSecret: os.Getenv("SECRET"),
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
	mux.HandleFunc("POST "+apiPath+"/login", apiCfg.userLogin)
	mux.HandleFunc("POST "+apiPath+"/refresh", apiCfg.TkHandlerRefresh)
	mux.HandleFunc("POST "+apiPath+"/revoke", apiCfg.revokeRefreshToken)
	mux.HandleFunc("PUT "+apiPath+"/users", apiCfg.updateUser)
	mux.HandleFunc("DELETE "+apiPath+"/chirps/{chirpID}", apiCfg.delChirpById)

	server := &http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}
	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}

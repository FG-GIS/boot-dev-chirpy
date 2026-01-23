package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
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
	res := []byte{}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(fmt.Appendf(res, "Hits reset from: %v\nTo: 0", cfg.fileserverHits.Swap(0)))
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

func validationHandler(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}
	type validated struct {
		ValidCheck bool `json:"valid"`
	}

	decoder := json.NewDecoder(r.Body)
	message := chirp{}
	err := decoder.Decode(&message)
	code := 200
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error decoding the message: %s", err))
	}
	respBody := validated{
		ValidCheck: true,
	}
	if len([]rune(message.Body)) > 140 {
		respBody.ValidCheck = false
		code = 400
		respondWithError(w, code, "Chirp is too long.")
	}
	respondWithJSON(w, code, respBody)
}

func main() {
	apiCfg := apiConfig{}
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
	mux.HandleFunc("POST "+apiPath+"/validate_chirp", validationHandler)

	server := &http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}
	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}

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

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/mikegmatthews/chirpy-http-server/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
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
	resp.Header().Add("Content-Type", "text/html")
	fmt.Fprintf(resp, `
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, hits)
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

func validateChirp(resp http.ResponseWriter, req *http.Request) {
	type validParams struct {
		Body string `json:"body"`
	}

	type validReturn struct {
		Valid       bool   `json:"valid"`
		CleanedBody string `json:"cleaned_body"`
	}

	var params validParams
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(resp, 500, fmt.Sprintf("Error decoding body JSON: %s\n", err))
		return
	}

	if len(params.Body) > 140 {
		respondWithError(resp, 400, "Chirp is too long")
		return
	}

	respondWithJSON(resp, 200, &validReturn{
		Valid:       true,
		CleanedBody: cleanChirp(params.Body),
	})
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type errorResponse struct {
		Error string `json:"error"`
	}

	w.WriteHeader(code)
	w.Header().Add("Content-Type", "application/json")
	resp := errorResponse{
		Error: msg,
	}
	bytes, err := json.Marshal(&resp)
	if err != nil {
		log.Printf("Error creating JSON error payload: %s\n", err)
	}
	w.Write(bytes)
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling JSON payload: %s\n", err)
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(bytes)
}

func cleanChirp(chirp string) string {
	words := strings.Split(chirp, " ")

	badWords := []string{
		"kerfuffle",
		"sharbert",
		"fornax",
	}

	for i, word := range words {
		if slices.Contains(badWords, strings.ToLower(word)) {
			words[i] = "****"
		}
	}

	return strings.Join(words, " ")
}

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening PostgreSQL connection: %s\n", err)
	}

	serveMux := http.NewServeMux()
	server := http.Server{
		Handler: serveMux,
		Addr:    ":8080",
	}

	conf := apiConfig{}
	conf.dbQueries = database.New(db)

	appHandler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))
	serveMux.Handle("/app/", conf.middlewareMetricsInc(appHandler))
	serveMux.HandleFunc("GET /api/healthz", healthStatus)
	serveMux.HandleFunc("GET /admin/metrics", conf.handleHitResponse)
	serveMux.HandleFunc("POST /admin/reset", conf.handleReset)
	serveMux.HandleFunc("POST /api/validate_chirp", validateChirp)

	log.Println("Starting HTTP server on port 8080")
	log.Fatal(server.ListenAndServe())
}

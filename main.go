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

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/mikegmatthews/chirpy-http-server/internal/auth"
	"github.com/mikegmatthews/chirpy-http-server/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
}

func (c *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	metricsInc := func(w http.ResponseWriter, r *http.Request) {
		c.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(metricsInc)
}

func (c *apiConfig) handleHitResponse(w http.ResponseWriter, r *http.Request) {
	hits := c.fileserverHits.Load()
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "text/html")
	fmt.Fprintf(w, `
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, hits)
}

func (c *apiConfig) handleReset(w http.ResponseWriter, r *http.Request) {
	platform := os.Getenv("PLATFORM")
	if platform != "dev" {
		respondWithError(w, 403, "FORBIDDEN")
		return
	}

	c.fileserverHits.Store(0)
	err := c.dbQueries.DeleteAllUsers(r.Context())
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error deleting all users: %s\n", err))
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Hits reset to 0")
}

func (c *apiConfig) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	type createUserParams struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	var newUserEmail createUserParams
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&newUserEmail)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError,
			fmt.Sprintf("Error decoding JSON: %s\n", err))
		return
	}

	if newUserEmail.Email == "" {
		respondWithError(w, http.StatusBadRequest, "User email cannot be blank")
		return
	} else if newUserEmail.Password == "" {
		respondWithError(w, http.StatusBadRequest, "User password cannot be blank")
		return
	}

	hash, err := auth.HashPassword(newUserEmail.Password)
	dbReturn, err := c.dbQueries.CreatUser(r.Context(), database.CreatUserParams{
		Email:          newUserEmail.Email,
		HashedPassword: hash,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError,
			fmt.Sprintf("Error creating new user: %s\n", err))
		return
	}

	newUser := User{
		ID:        dbReturn.ID,
		CreatedAt: dbReturn.CreatedAt,
		UpdatedAt: dbReturn.UpdatedAt,
		Email:     dbReturn.Email,
	}
	respondWithJSON(w, http.StatusCreated, newUser)
}

func (c *apiConfig) handleLogin(w http.ResponseWriter, r *http.Request) {
	type loginParams struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	var newLogin loginParams
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&newLogin)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError,
			fmt.Sprintf("Error decoding JSON: %s\n", err))
		return
	}

	dbUser, err := c.dbQueries.GetUserByEmail(r.Context(), newLogin.Email)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}
	success, err := auth.CheckPasswordHash(newLogin.Password, dbUser.HashedPassword)
	if err != nil || !success {
		respondWithError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	loggedInUser := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}
	respondWithJSON(w, http.StatusOK, loggedInUser)
}

func (c *apiConfig) handleCreateChirp(w http.ResponseWriter, r *http.Request) {
	type chirpReq struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	var params chirpReq
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error decoding body JSON: %s\n", err))
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	dbReturn, err := c.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   params.Body,
		UserID: params.UserID,
	})
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error creating new chirp: %s\n", err))
		return
	}

	newChirp := Chirp{
		ID:        dbReturn.ID,
		CreatedAt: dbReturn.CreatedAt,
		UpdatedAt: dbReturn.UpdatedAt,
		Body:      dbReturn.Body,
		UserID:    dbReturn.UserID,
	}
	respondWithJSON(w, 201, newChirp)
}

func (c *apiConfig) handleGetAllChirps(w http.ResponseWriter, r *http.Request) {
	dbChirps, err := c.dbQueries.GetAllChirps(r.Context())
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error getting all chirps: %s\n", err))
		return
	}

	allChirps := make([]Chirp, len(dbChirps))
	for i, chirp := range dbChirps {
		allChirps[i] = Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}
	}

	respondWithJSON(w, 200, allChirps)
}

func (c *apiConfig) handleGetChirp(w http.ResponseWriter, r *http.Request) {
	chirpID := r.PathValue("chirpID")
	chirpUUID, err := uuid.Parse(chirpID)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Error parsing UUID: %s\n", err))
		return
	}

	dbChirp, err := c.dbQueries.GetChirp(r.Context(), chirpUUID)
	if err != nil {
		respondWithError(w, 404, "CHIRP NOT FOUND")
		return
	}

	chirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	respondWithJSON(w, 200, chirp)
}

func healthStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
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
	serveMux.HandleFunc("POST /api/chirps", conf.handleCreateChirp)
	serveMux.HandleFunc("GET /api/chirps", conf.handleGetAllChirps)
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", conf.handleGetChirp)
	serveMux.HandleFunc("POST /api/users", conf.handleCreateUser)
	serveMux.HandleFunc("POST /api/login", conf.handleLogin)

	log.Println("Starting HTTP server on port 8080")
	log.Fatal(server.ListenAndServe())
}

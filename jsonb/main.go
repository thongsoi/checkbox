package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	_ "github.com/lib/pq"
)

// Database connection
var db *sql.DB

// Structs for handling requests and responses
type PreferencesRequest struct {
	Preferences []string `json:"preferences"`
}

type PreferencesResponse struct {
	ID          int      `json:"id"`
	UserID      int      `json:"user_id"`
	Preferences []string `json:"preferences"`
	CreatedAt   string   `json:"created_at"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Database initialization
func initDB() {
	var err error
	// Replace with your actual database connection string
	connStr := "user=postgres dbname=abirtek sslmode=disable password=440 host=localhost"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Create table if it doesn't exist
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS user_preferences (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL,
		preferences JSONB,
		created_at TIMESTAMP DEFAULT NOW()
	);
	
	-- Create index for better JSONB performance
	CREATE INDEX IF NOT EXISTS idx_user_preferences_jsonb 
	ON user_preferences USING GIN (preferences);
	`

	if _, err := db.Exec(createTableQuery); err != nil {
		log.Fatal("Failed to create table:", err)
	}
}

// Handler to save user preferences
func savePreferencesHandler(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		render.JSON(w, r, ErrorResponse{Error: "User ID is required"})
		return
	}

	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		render.JSON(w, r, ErrorResponse{Error: "Invalid user ID"})
		return
	}

	var req PreferencesRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		render.JSON(w, r, ErrorResponse{Error: "Invalid JSON payload"})
		return
	}

	// Convert preferences slice to JSON
	preferencesJSON, err := json.Marshal(req.Preferences)
	if err != nil {
		render.JSON(w, r, ErrorResponse{Error: "Failed to process preferences"})
		return
	}

	// Insert or update preferences in database
	query := `
	INSERT INTO user_preferences (user_id, preferences) 
	VALUES ($1, $2)
	ON CONFLICT (user_id) DO UPDATE SET 
		preferences = $2,
		created_at = NOW()
	RETURNING id, created_at
	`

	var id int
	var createdAt string
	err = db.QueryRow(query, userIDInt, preferencesJSON).Scan(&id, &createdAt)
	if err != nil {
		log.Printf("Database error: %v", err)
		render.JSON(w, r, ErrorResponse{Error: "Failed to save preferences"})
		return
	}

	response := PreferencesResponse{
		ID:          id,
		UserID:      userIDInt,
		Preferences: req.Preferences,
		CreatedAt:   createdAt,
	}

	render.JSON(w, r, response)
}

// Handler to get user preferences
func getPreferencesHandler(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		render.JSON(w, r, ErrorResponse{Error: "User ID is required"})
		return
	}

	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		render.JSON(w, r, ErrorResponse{Error: "Invalid user ID"})
		return
	}

	query := `
	SELECT id, user_id, preferences, created_at 
	FROM user_preferences 
	WHERE user_id = $1
	`

	var id int
	var uid int
	var preferencesJSON []byte
	var createdAt string

	err = db.QueryRow(query, userIDInt).Scan(&id, &uid, &preferencesJSON, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty preferences if no record found
			response := PreferencesResponse{
				UserID:      userIDInt,
				Preferences: []string{},
			}
			render.JSON(w, r, response)
			return
		}
		log.Printf("Database error: %v", err)
		render.JSON(w, r, ErrorResponse{Error: "Failed to fetch preferences"})
		return
	}

	// Parse JSONB data
	var preferences []string
	if err := json.Unmarshal(preferencesJSON, &preferences); err != nil {
		log.Printf("JSON unmarshal error: %v", err)
		render.JSON(w, r, ErrorResponse{Error: "Failed to parse preferences"})
		return
	}

	response := PreferencesResponse{
		ID:          id,
		UserID:      uid,
		Preferences: preferences,
		CreatedAt:   createdAt,
	}

	render.JSON(w, r, response)
}

// Handler to search users by specific preference
func searchByPreferenceHandler(w http.ResponseWriter, r *http.Request) {
	preference := r.URL.Query().Get("preference")
	if preference == "" {
		render.JSON(w, r, ErrorResponse{Error: "Preference parameter is required"})
		return
	}

	// Use JSONB operators to search for users with specific preference
	query := `
	SELECT id, user_id, preferences, created_at 
	FROM user_preferences 
	WHERE preferences ? $1
	`

	rows, err := db.Query(query, preference)
	if err != nil {
		log.Printf("Database error: %v", err)
		render.JSON(w, r, ErrorResponse{Error: "Failed to search preferences"})
		return
	}
	defer rows.Close()

	var results []PreferencesResponse
	for rows.Next() {
		var id, uid int
		var preferencesJSON []byte
		var createdAt string

		if err := rows.Scan(&id, &uid, &preferencesJSON, &createdAt); err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}

		var preferences []string
		if err := json.Unmarshal(preferencesJSON, &preferences); err != nil {
			log.Printf("JSON unmarshal error: %v", err)
			continue
		}

		results = append(results, PreferencesResponse{
			ID:          id,
			UserID:      uid,
			Preferences: preferences,
			CreatedAt:   createdAt,
		})
	}

	render.JSON(w, r, results)
}

// Setup routes
func setupRoutes() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	// CORS middleware for frontend integration
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Routes
	r.Route("/api", func(r chi.Router) {
		r.Post("/preferences/{userID}", savePreferencesHandler)
		r.Get("/preferences/{userID}", getPreferencesHandler)
		r.Get("/search/preferences", searchByPreferenceHandler)
	})

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, map[string]string{"status": "ok"})
	})

	return r
}

func main() {
	// Initialize database
	initDB()
	defer db.Close()

	// Setup routes
	router := setupRoutes()

	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

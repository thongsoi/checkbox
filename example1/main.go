package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

var db *sql.DB

// Replace with your actual PostgreSQL connection info
const (
	DB_USER     = "postgres"
	DB_PASSWORD = "440"
	DB_NAME     = "abirtek"
	DB_HOST     = "localhost"
	DB_PORT     = "5432"
)

func main() {
	// Connect to PostgreSQL
	var err error
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)

	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal("Database ping failed:", err)
	}

	// Create table if not exists
	createTable()

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Serve static HTML
	r.Get("/", serveForm)

	// Handle checkbox POST
	r.Post("/api/save-checkboxes", saveCheckboxesHandler)

	fmt.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// Struct for parsing JSON
type CheckboxRequest struct {
	Selected []string `json:"selected"`
}

// Handler to serve the checkbox HTML form
func serveForm(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head><title>Checkbox Form</title></head>
<body>
  <form id="checkboxForm">
    <label><input type="checkbox" name="options" value="option1"> Option 1</label><br>
    <label><input type="checkbox" name="options" value="option2"> Option 2</label><br>
    <label><input type="checkbox" name="options" value="option3"> Option 3</label><br>
    <button type="submit">Submit</button>
  </form>

  <script>
    document.getElementById("checkboxForm").addEventListener("submit", async function(event) {
      event.preventDefault();
      const checked = Array.from(document.querySelectorAll('input[name="options"]:checked'))
                          .map(cb => cb.value);

      const res = await fetch("/api/save-checkboxes", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ selected: checked })
      });

      if (res.ok) {
        alert("Checkboxes saved!");
      } else {
        alert("Error saving checkboxes.");
      }
    });
  </script>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// Handler to store checkbox values
func saveCheckboxesHandler(w http.ResponseWriter, r *http.Request) {
	var req CheckboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	query := `INSERT INTO user_selections (option_name) VALUES ($1)`
	for _, option := range req.Selected {
		if _, err := db.Exec(query, option); err != nil {
			log.Println("DB insert error:", err)
			http.Error(w, "Failed to insert", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Saved"))
}

// Helper to create the table
func createTable() {
	query := `
	CREATE TABLE IF NOT EXISTS user_selections (
		id SERIAL PRIMARY KEY,
		option_name TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := db.Exec(query); err != nil {
		log.Fatal("Failed to create table:", err)
	}
}

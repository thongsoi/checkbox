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

// DB configuration (replace with your actual credentials)
const (
	DB_USER     = "postgres"
	DB_PASSWORD = "440"
	DB_NAME     = "abirtek"
	DB_HOST     = "localhost"
	DB_PORT     = "5432"
)

func main() {
	var err error
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)

	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("Error connecting to DB:", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("Database unreachable:", err)
	}

	createTable()

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/", serveForm)
	r.Post("/api/save-checkboxes", saveCheckboxesHandler)

	fmt.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

// Struct to parse JSON from frontend
type CheckboxRequest struct {
	UserID   int      `json:"user_id"`
	Selected []string `json:"selected"`
}

// Serve HTML form
func serveForm(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head><title>Checkbox Form</title></head>
<body>
  <h2>Select Your Options</h2>
  <form id="checkboxForm">
    <input type="hidden" id="userId" value="123"> <!-- This would usually come from your login system -->
    <label><input type="checkbox" name="options" value="option1"> Option 1</label><br>
    <label><input type="checkbox" name="options" value="option2"> Option 2</label><br>
    <label><input type="checkbox" name="options" value="option3"> Option 3</label><br>
    <button type="submit">Submit</button>
  </form>

  <script>
    document.getElementById("checkboxForm").addEventListener("submit", async function(event) {
      event.preventDefault();

      const userId = parseInt(document.getElementById("userId").value);
      const checked = Array.from(document.querySelectorAll('input[name="options"]:checked'))
                          .map(cb => cb.value);

      const res = await fetch("/api/save-checkboxes", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          user_id: userId,
          selected: checked
        })
      });

      if (res.ok) {
        alert("Selections saved!");
      } else {
        alert("Failed to save.");
      }
    });
  </script>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// Save checkbox selections with user_id
func saveCheckboxesHandler(w http.ResponseWriter, r *http.Request) {
	var req CheckboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	query := `INSERT INTO user_selections (user_id, option_name) VALUES ($1, $2)`

	for _, option := range req.Selected {
		if _, err := db.Exec(query, req.UserID, option); err != nil {
			log.Println("Insert error:", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Saved"))
}

// Create table if not exists
func createTable() {
	query := `
	CREATE TABLE IF NOT EXISTS user_selections (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL,
		option_name TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	if _, err := db.Exec(query); err != nil {
		log.Fatal("Failed to create table:", err)
	}
}

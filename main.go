package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Session struct {
	Code      string
	Submitted bool
}

var (
	db        *sql.DB
	tmpl      *template.Template
	questText string
)

type PageData struct {
	UUID      string
	Quest     string
	Code      string
	Submitted bool
}

func main() {
	// Load the challenge text from file.
	var err error
	questText, err = loadQuest("quest.txt")
	if err != nil {
		log.Fatalf("Failed to load quest: %v", err)
	}

	// Open (or create) the SQLite database.
	db, err = sql.Open("sqlite3", "./sessions.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create the sessions table if it does not exist.
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS sessions (
            uuid TEXT PRIMARY KEY,
            code TEXT,
            submitted BOOLEAN
        )
    `)
	if err != nil {
		log.Fatalf("Failed to create sessions table: %v", err)
	}

	// Parse the HTML template.
	tmpl = template.Must(template.ParseFiles("template.html"))

	// Use sessionHandler for all requests.
	http.HandleFunc("/", sessionHandler)
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func loadQuest(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func getSession(uuid string) (*Session, error) {
	row := db.QueryRow("SELECT code, submitted FROM sessions WHERE uuid = ?", uuid)
	var code string
	var submitted bool
	err := row.Scan(&code, &submitted)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Session{Code: code, Submitted: submitted}, nil
}

func createSession(uuid string) (*Session, error) {
	_, err := db.Exec(
		"INSERT INTO sessions (uuid, code, submitted) VALUES (?, ?, ?)",
		uuid,
		"",
		false,
	)
	if err != nil {
		return nil, err
	}
	return &Session{Code: "", Submitted: false}, nil
}

func updateSession(uuid string, session *Session) error {
	_, err := db.Exec(
		"UPDATE sessions SET code = ?, submitted = ? WHERE uuid = ?",
		session.Code,
		session.Submitted,
		uuid,
	)
	return err
}

// sessionHandler handles two routes:
//
//	GET /{uuid}         => render the page with the editor
//	POST /{uuid}/submit => update session with submitted code and mark as submitted
func sessionHandler(w http.ResponseWriter, r *http.Request) {
	// Remove leading/trailing slashes and split the path.
	path := strings.Trim(r.URL.Path, "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(path, "/")
	uuid := parts[0]

	// Load existing session or create a new one.
	session, err := getSession(uuid)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if session == nil {
		session, err = createSession(uuid)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	// Render full page on GET.
	if len(parts) == 1 && r.Method == http.MethodGet {
		renderPage(w, uuid, session)
		return
	}

	// Handle submission.
	if len(parts) == 2 && parts[1] == "submit" && r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Parse error", http.StatusBadRequest)
			return
		}
		code := r.FormValue("code")
		session.Code = code
		session.Submitted = true
		if err := updateSession(uuid, session); err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		renderPage(w, uuid, session)
		return
	}

	http.NotFound(w, r)
}

func renderPage(w http.ResponseWriter, uuid string, session *Session) {
	data := PageData{
		UUID:      uuid,
		Quest:     questText,
		Code:      session.Code,
		Submitted: session.Submitted,
	}
	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

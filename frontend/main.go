package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Message struct {
	Title    string `json:"title"`
	Message  string `json:"message"`
	Database bool   `json:"database"`
}

type PageData struct {
	Title    string
	Message  string
	Backend  string
	Error    string
	Database bool
	Notice   string
	Result   string
	Term     string
}

var httpClient = &http.Client{Timeout: 5 * time.Second}

func main() {
	tmpl := template.Must(template.ParseFiles("index.html.tmpl"))
	backendURL := envOr("BACKEND_URL", "http://test-app-backend:8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler(tmpl, backendURL))
	mux.HandleFunc("/add", addHandler(tmpl, backendURL))
	mux.HandleFunc("/search", searchHandler(tmpl, backendURL))
	mux.HandleFunc("/healthz", healthHandler)

	addr := ":" + envOr("PORT", "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("frontend listening on http://localhost%s (backend %s)", addr, backendURL)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func homeHandler(tmpl *template.Template, backendURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		render(tmpl, w, backendURL, nil)
	}
}

func addHandler(tmpl *template.Template, backendURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimSpace(r.PostFormValue("name"))
		notice := postEntry(backendURL, name)
		render(tmpl, w, backendURL, func(d *PageData) { d.Notice = notice })
	}
}

func searchHandler(tmpl *template.Template, backendURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimSpace(r.URL.Query().Get("name"))
		result := searchEntry(backendURL, name)
		render(tmpl, w, backendURL, func(d *PageData) {
			d.Term = name
			d.Result = result
		})
	}
}

func render(tmpl *template.Template, w http.ResponseWriter, backendURL string, extra func(*PageData)) {
	data := PageData{Title: "Hello World", Backend: backendURL}
	if msg, err := fetchMessage(backendURL); err != nil {
		data.Error = err.Error()
	} else {
		data.Title = msg.Title
		data.Message = msg.Message
		data.Database = msg.Database
	}
	if extra != nil {
		extra(&data)
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func postEntry(backendURL, name string) string {
	body, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		return "Could not send that value to the backend."
	}
	resp, err := httpClient.Post(backendURL+"/api/entries", "application/json", bytes.NewReader(body))
	if err != nil {
		return "Could not reach the backend."
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		return fmt.Sprintf("Saved %q to the database.", name)
	case http.StatusBadRequest:
		return "Enter a value between 1 and 100 characters."
	case http.StatusServiceUnavailable:
		return "No database is configured."
	default:
		return "Could not save that value."
	}
}

func searchEntry(backendURL, name string) string {
	resp, err := httpClient.Get(backendURL + "/api/entries/search?name=" + url.QueryEscape(name))
	if err != nil {
		return "Could not reach the backend."
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusBadRequest:
		return "Enter a value between 1 and 100 characters."
	case http.StatusServiceUnavailable:
		return "No database is configured."
	default:
		return "Could not search for that value."
	}

	var out struct {
		Exists bool `json:"exists"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "Could not read the search result."
	}
	if out.Exists {
		return "This exists in the database"
	}
	return "Sorry, this does not exist in the database"
}

func fetchMessage(backendURL string) (Message, error) {
	var msg Message
	resp, err := httpClient.Get(backendURL + "/api/message")
	if err != nil {
		return msg, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return msg, fmt.Errorf("backend returned status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		return msg, err
	}
	return msg, nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

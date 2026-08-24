package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"
)

type Message struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

type PageData struct {
	Title   string
	Message string
	Backend string
	Error   string
}

var httpClient = &http.Client{Timeout: 5 * time.Second}

func main() {
	tmpl := template.Must(template.ParseFiles("index.html.tmpl"))
	backendURL := envOr("BACKEND_URL", "http://test-app-backend:8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler(tmpl, backendURL))
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
		data := PageData{Title: "Hello World", Backend: backendURL}
		if msg, err := fetchMessage(backendURL); err != nil {
			data.Error = err.Error()
		} else {
			data.Title = msg.Title
			data.Message = msg.Message
		}
		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
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

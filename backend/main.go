package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type Message struct {
	Title    string `json:"title"`
	Message  string `json:"message"`
	Database bool   `json:"database"`
}

var payload = Message{
	Title:   "Hello World",
	Message: "Welcome to the container platform (v2)",
}

func main() {
	ctx := context.Background()

	db, err := openStore(ctx)
	if err != nil {
		log.Fatalf("database setup failed: %v", err)
	}
	defer db.close()
	if db == nil {
		log.Print("PGHOST not set, database features disabled")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/message", messageHandler(db != nil))
	mux.HandleFunc("/api/entries", addHandler(db))
	mux.HandleFunc("/api/entries/search", searchHandler(db))
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
	log.Printf("backend listening on http://localhost%s", addr)
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

func messageHandler(dbEnabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body := payload
		body.Database = dbEnabled
		writeJSON(w, http.StatusOK, body)
	}
}

func addHandler(db *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if db == nil {
			http.Error(w, "Database Not Configured", http.StatusServiceUnavailable)
			return
		}

		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		name, ok := validName(req.Name)
		if !ok {
			http.Error(w, "Name must be between 1 and 100 characters", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if err := db.add(ctx, name); err != nil {
			log.Printf("insert failed: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"saved": name})
	}
}

func searchHandler(db *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			http.Error(w, "Database Not Configured", http.StatusServiceUnavailable)
			return
		}

		name, ok := validName(r.URL.Query().Get("name"))
		if !ok {
			http.Error(w, "Name must be between 1 and 100 characters", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		found, err := db.exists(ctx, name)
		if err != nil {
			log.Printf("search failed: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"name": name, "exists": found})
	}
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("encode failed: %v", err)
	}
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

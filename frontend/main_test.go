package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "ok")
	}
}

func TestFetchMessage(t *testing.T) {
	want := Message{Title: "T", Message: "M"}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/message" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer backend.Close()

	got, err := fetchMessage(backend.URL)
	if err != nil {
		t.Fatalf("fetchMessage: %v", err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestFetchMessageBackendError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer backend.Close()

	if _, err := fetchMessage(backend.URL); err == nil {
		t.Fatal("expected error for non-200 backend, got nil")
	}
}

func TestHomeHandler(t *testing.T) {
	want := Message{Title: "Hello", Message: "World"}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer backend.Close()

	tmpl := template.Must(template.New("index.html.tmpl").Parse(`{{.Title}}|{{.Message}}|{{.Backend}}|{{.Error}}`))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	homeHandler(tmpl, backend.URL)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Hello") || !strings.Contains(body, "World") {
		t.Errorf("body %q missing rendered message", body)
	}
}

func TestHomeHandlerNotFound(t *testing.T) {
	tmpl := template.Must(template.New("index.html.tmpl").Parse(`{{.Title}}`))
	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	rec := httptest.NewRecorder()

	homeHandler(tmpl, "http://127.0.0.1:0")(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

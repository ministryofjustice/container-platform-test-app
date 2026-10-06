package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMessageHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/message", nil)
	rec := httptest.NewRecorder()

	messageHandler(false)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var got Message
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got != payload {
		t.Errorf("body = %+v, want %+v", got, payload)
	}
}

func TestMessageHandlerReportsDatabaseEnabled(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/message", nil)
	rec := httptest.NewRecorder()

	messageHandler(true)(rec, req)

	var got Message
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !got.Database {
		t.Error("Database = false, want true")
	}
}

func TestValidName(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		want  string
		valid bool
	}{
		{"trims surrounding space", "  alice  ", "alice", true},
		{"rejects empty", "", "", false},
		{"rejects whitespace only", "   ", "", false},
		{"rejects overlong", strings.Repeat("a", maxNameLen+1), "", false},
		{"accepts max length", strings.Repeat("a", maxNameLen), strings.Repeat("a", maxNameLen), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := validName(tc.in)
			if ok != tc.valid {
				t.Fatalf("ok = %v, want %v", ok, tc.valid)
			}
			if got != tc.want {
				t.Errorf("name = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHandlersWithoutDatabase(t *testing.T) {
	body := strings.NewReader(`{"name":"alice"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/entries", body)
	rec := httptest.NewRecorder()
	addHandler(nil)(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("add status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/entries/search?name=alice", nil)
	rec = httptest.NewRecorder()
	searchHandler(nil)(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("search status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestAddHandlerRejectsGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/entries", nil)
	rec := httptest.NewRecorder()

	addHandler(nil)(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

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

func TestSecurityHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	securityHeaders(next).ServeHTTP(rec, req)

	want := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	}
	for k, v := range want {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
}

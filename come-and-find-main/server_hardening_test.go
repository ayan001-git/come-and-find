package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRejectUnknownJSONFields(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"A","unexpected":1}`))
	w := httptest.NewRecorder()
	var input struct {
		Name string `json:"name"`
	}
	if decodeJSON(w, r, &input) {
		t.Fatal("unknown field must be rejected")
	}
	if w.Code != 400 {
		t.Fatalf("want 400, got %d", w.Code)
	}
	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
}

func TestSecurityHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	w := httptest.NewRecorder()
	sameOrigin(next).ServeHTTP(w, httptest.NewRequest("GET", "http://localhost:8080/", nil))
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("missing frame protection")
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff")
	}
}

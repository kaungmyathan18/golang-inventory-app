package apiresponse

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
)

func TestWriteJSONWritesEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()

	handler := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, r, http.StatusCreated, map[string]string{"id": "user-1"}, &PaginationMeta{
			Page:    2,
			Limit:   10,
			Total:   25,
			HasMore: true,
		}, &Links{Self: "http://example.com/api/v1/users?page=2&limit=10"})
	}))
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/users", nil))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type = %q, want application/json", got)
	}

	var body Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Meta.RequestID == "" {
		t.Fatal("request id is empty")
	}
	if body.Meta.APIVersion != "v1" {
		t.Fatalf("api version = %q, want v1", body.Meta.APIVersion)
	}
	if body.Meta.Pagination == nil || !body.Meta.Pagination.HasMore {
		t.Fatalf("pagination = %#v, want populated has_more", body.Meta.Pagination)
	}
	if body.Links == nil || body.Links.Self == "" {
		t.Fatalf("links = %#v, want self link", body.Links)
	}
}

func TestWriteProblemUsesForwardedProtoAndRequestContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://api.test.local/users", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()

	handler := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteProblem(w, r, http.StatusUnprocessableEntity, ProblemTypeURI(r, "validation"), "Validation Failed", "bad input", []FieldProblem{
			{Field: "email", Message: "required", Code: "REQUIRED"},
		})
	}))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("content type = %q, want application/problem+json", got)
	}

	var body Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Type != "https://api.test.local/problems/validation" {
		t.Fatalf("type = %q", body.Type)
	}
	if body.Instance != "/users" {
		t.Fatalf("instance = %q, want /users", body.Instance)
	}
	if body.RequestID == "" {
		t.Fatal("request id is empty")
	}
	if len(body.Errors) != 1 || body.Errors[0].Field != "email" {
		t.Fatalf("errors = %#v, want email field problem", body.Errors)
	}
}

func TestPageLinks(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/users?filter=active", nil)

	links := PageLinks(req, 2, 10, true)

	if links.Self != "http://example.com/api/v1/users?filter=active&limit=10&page=2" {
		t.Fatalf("self = %q", links.Self)
	}
	if links.Next != "http://example.com/api/v1/users?filter=active&limit=10&page=3" {
		t.Fatalf("next = %q", links.Next)
	}
	if links.Prev != "http://example.com/api/v1/users?filter=active&limit=10&page=1" {
		t.Fatalf("prev = %q", links.Prev)
	}
}

func TestWriteInternalErrorAndURLHelpers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://secure.example.com/fail", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()

	WriteInternalError(rec, req, errors.New("database down"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var body Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if body.Type != "https://secure.example.com/problems/internal-error" {
		t.Fatalf("type = %q", body.Type)
	}
	if got := instancePath(&http.Request{}); got != "/" {
		t.Fatalf("instancePath without URL = %q, want /", got)
	}

	relativeReq := httptest.NewRequest(http.MethodGet, "/items?page=1", nil)
	relativeReq.Host = "relative.example.com"
	relativeReq.Header.Set("X-Forwarded-Proto", "https")
	links := PageLinks(relativeReq, 1, 20, false)
	if links.Self != "https://relative.example.com/items?limit=20&page=1" {
		t.Fatalf("relative self = %q", links.Self)
	}
}

package validation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kaungmyathan18/golang-inventory-app/internal/apiresponse"
)

type decodeTestPayload struct {
	Email string `json:"email" validate:"required,email"`
	Count int    `json:"count" validate:"gte=1,lte=10"`
}

func TestDecodeJSONValidatesAndRejectsUnknownFields(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"test@example.com","count":3}`))
		var payload decodeTestPayload

		if err := DecodeJSON(req, &payload); err != nil {
			t.Fatalf("DecodeJSON returned error: %v", err)
		}
		if payload.Email != "test@example.com" || payload.Count != 3 {
			t.Fatalf("payload = %#v", payload)
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"test@example.com","count":3,"extra":true}`))
		var payload decodeTestPayload

		if err := DecodeJSON(req, &payload); err == nil {
			t.Fatal("DecodeJSON returned nil error, want unknown field error")
		}
	})

	t.Run("validation error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"bad","count":0}`))
		var payload decodeTestPayload

		if err := DecodeJSON(req, &payload); err == nil {
			t.Fatal("DecodeJSON returned nil error, want validation error")
		}
	})
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantTitle  string
		wantErrors bool
	}{
		{
			name:       "invalid json",
			body:       `{"email":`,
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Invalid JSON",
		},
		{
			name:       "type mismatch",
			body:       `{"email":"test@example.com","count":"nope"}`,
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Invalid JSON",
		},
		{
			name:       "validation",
			body:       `{"email":"","count":0}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantTitle:  "Validation Failed",
			wantErrors: true,
		},
		{
			name:       "unknown field",
			body:       `{"email":"test@example.com","count":3,"extra":true}`,
			wantStatus: http.StatusBadRequest,
			wantTitle:  "Bad Request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://example.com/resource", strings.NewReader(tt.body))
			var payload decodeTestPayload
			err := DecodeJSON(req, &payload)
			if err == nil {
				t.Fatal("DecodeJSON returned nil error")
			}

			rec := httptest.NewRecorder()
			WriteError(rec, req, err)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var problem apiresponse.Problem
			if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
				t.Fatalf("decode problem: %v", err)
			}
			if problem.Title != tt.wantTitle {
				t.Fatalf("title = %q, want %q", problem.Title, tt.wantTitle)
			}
			if tt.wantErrors && len(problem.Errors) == 0 {
				t.Fatalf("errors = %#v, want validation errors", problem.Errors)
			}
		})
	}
}

func TestVar(t *testing.T) {
	if err := Var("550e8400-e29b-41d4-a716-446655440000", "required,uuid"); err != nil {
		t.Fatalf("valid uuid returned error: %v", err)
	}
	if err := Var("not-a-uuid", "required,uuid"); err == nil {
		t.Fatal("invalid uuid returned nil error")
	}
}

func TestValidationErrorFormattingHelpers(t *testing.T) {
	payload := struct {
		Name      string `json:"name" validate:"min=3,max=5"`
		Status    string `json:"status" validate:"oneof=open closed"`
		Code      string `json:"code" validate:"len=4"`
		StartedAt string `json:"started_at" validate:"datetime=2006-01-02"`
		Prefix    string `json:"prefix" validate:"startswith=INV"`
	}{Name: "xy", Status: "bad", Code: "123", StartedAt: "bad-date", Prefix: "bad"}

	err := V.Struct(payload)
	if err == nil {
		t.Fatal("Struct returned nil error")
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example.com/items", nil)
	WriteError(rec, req, err)

	var problem apiresponse.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if len(problem.Errors) != 5 {
		t.Fatalf("errors = %#v, want 5", problem.Errors)
	}
	seen := map[string]bool{}
	for _, fieldErr := range problem.Errors {
		seen[fieldErr.Code] = true
		if fieldErr.Field == "" || fieldErr.Message == "" {
			t.Fatalf("field error missing content: %#v", fieldErr)
		}
	}
	for _, code := range []string{"OUT_OF_RANGE", "INVALID_VALUE", "INVALID_FORMAT", "INVALID"} {
		if !seen[code] {
			t.Fatalf("did not see code %s in %#v", code, problem.Errors)
		}
	}
}

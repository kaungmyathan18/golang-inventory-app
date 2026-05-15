//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type envelope[T any] struct {
	Data T `json:"data"`
	Meta struct {
		APIVersion string `json:"api_version"`
	} `json:"meta"`
}

type problem struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

type category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type product struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	CategoryID string  `json:"category_id"`
}

type inventory struct {
	ID        string `json:"id"`
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

func TestInventoryAPIEndToEnd(t *testing.T) {
	baseURL := strings.TrimRight(os.Getenv("E2E_BASE_URL"), "/")
	if baseURL == "" {
		t.Skip("E2E_BASE_URL is not set")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	waitReady(t, client, baseURL)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	cat := doJSON[category](t, client, http.MethodPost, baseURL+"/api/v1/categories/", map[string]any{
		"name": "E2E Category " + suffix,
	}, http.StatusCreated)
	if cat.ID == "" {
		t.Fatal("category id is empty")
	}

	missingRelation := doProblem(t, client, http.MethodPost, baseURL+"/api/v1/products/", map[string]any{
		"name":        "E2E Missing Category Product " + suffix,
		"description": "negative smoke",
		"price":       1.25,
		"category_id": "550e8400-e29b-41d4-a716-446655440000",
	}, http.StatusUnprocessableEntity)
	if missingRelation.Title != "Validation Failed" {
		t.Fatalf("missing relation title = %q", missingRelation.Title)
	}

	prod := doJSON[product](t, client, http.MethodPost, baseURL+"/api/v1/products/", map[string]any{
		"name":        "E2E Product " + suffix,
		"description": "created by E2E test",
		"price":       42.5,
		"category_id": cat.ID,
	}, http.StatusCreated)
	if prod.ID == "" || prod.CategoryID != cat.ID {
		t.Fatalf("product = %#v", prod)
	}

	inv := doJSON[inventory](t, client, http.MethodPost, baseURL+"/api/v1/inventories/", map[string]any{
		"product_id": prod.ID,
		"quantity":   9,
	}, http.StatusCreated)
	if inv.ID == "" || inv.ProductID != prod.ID {
		t.Fatalf("inventory = %#v", inv)
	}

	gotProduct := doJSON[product](t, client, http.MethodGet, baseURL+"/api/v1/products/"+prod.ID, nil, http.StatusOK)
	if gotProduct.Name != prod.Name {
		t.Fatalf("got product = %#v", gotProduct)
	}
	gotInventory := doJSON[inventory](t, client, http.MethodGet, baseURL+"/api/v1/inventories/"+inv.ID, nil, http.StatusOK)
	if gotInventory.Quantity != 9 {
		t.Fatalf("got inventory = %#v", gotInventory)
	}

	_ = doJSON[[]category](t, client, http.MethodGet, baseURL+"/api/v1/categories/?page=1&limit=20", nil, http.StatusOK)
	_ = doJSON[[]product](t, client, http.MethodGet, baseURL+"/api/v1/products/?page=1&limit=20", nil, http.StatusOK)
	_ = doJSON[[]inventory](t, client, http.MethodGet, baseURL+"/api/v1/inventories/?page=1&limit=20", nil, http.StatusOK)

	invalidUUID := doProblem(t, client, http.MethodGet, baseURL+"/api/v1/products/not-a-uuid", nil, http.StatusUnprocessableEntity)
	if invalidUUID.Title != "Validation Failed" {
		t.Fatalf("invalid uuid title = %q", invalidUUID.Title)
	}

	doJSON[any](t, client, http.MethodDelete, baseURL+"/api/v1/inventories/"+inv.ID, nil, http.StatusOK)
	doJSON[any](t, client, http.MethodDelete, baseURL+"/api/v1/products/"+prod.ID, nil, http.StatusOK)
	doJSON[any](t, client, http.MethodDelete, baseURL+"/api/v1/categories/"+cat.ID, nil, http.StatusOK)
}

func waitReady(t *testing.T, client *http.Client, baseURL string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/health/ready", nil)
		if err != nil {
			t.Fatalf("build ready request: %v", err)
		}
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}

		select {
		case <-ctx.Done():
			t.Fatalf("app did not become ready: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}

func doJSON[T any](t *testing.T, client *http.Client, method, url string, payload any, wantStatus int) T {
	t.Helper()

	body := doRequest(t, client, method, url, payload, wantStatus)
	var env envelope[T]
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode envelope: %v; body=%s", err, string(body))
	}
	if env.Meta.APIVersion != "v1" {
		t.Fatalf("api version = %q", env.Meta.APIVersion)
	}
	return env.Data
}

func doProblem(t *testing.T, client *http.Client, method, url string, payload any, wantStatus int) problem {
	t.Helper()

	body := doRequest(t, client, method, url, payload, wantStatus)
	var p problem
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatalf("decode problem: %v; body=%s", err, string(body))
	}
	if p.Status != wantStatus {
		t.Fatalf("problem status = %d, want %d", p.Status, wantStatus)
	}
	return p
}

func doRequest(t *testing.T, client *http.Client, method, url string, payload any, wantStatus int) []byte {
	t.Helper()

	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("encode payload: %v", err)
		}
	}
	req, err := http.NewRequest(method, url, &body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	var responseBody bytes.Buffer
	if _, err := responseBody.ReadFrom(resp.Body); err != nil {
		t.Fatalf("read response: %v", err)
	}
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, url, resp.StatusCode, wantStatus, responseBody.String())
	}
	return responseBody.Bytes()
}

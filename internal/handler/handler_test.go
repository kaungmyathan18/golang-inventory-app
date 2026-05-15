package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kaungmyathan18/golang-inventory-app/internal/apiresponse"
	"github.com/kaungmyathan18/golang-inventory-app/internal/config"
	"github.com/kaungmyathan18/golang-inventory-app/internal/database"
	"github.com/kaungmyathan18/golang-inventory-app/internal/repository"
	"github.com/kaungmyathan18/golang-inventory-app/internal/service"
	"go.uber.org/zap"
)

type testApp struct {
	router http.Handler
	db     *database.DB
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()

	db, err := database.New(context.Background(), config.DatabaseConfig{
		Driver:          "sqlite",
		DSN:             "file:" + filepath.Join(t.TempDir(), "test.db") + "?_foreign_keys=on",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
	})
	if err := database.RunMigrations(context.Background(), db, migrationsPath(t)); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	log := zap.NewNop()
	userSvc := service.NewUserService(repository.NewUserRepository(db, log), log, nil)
	categoryRepo := repository.NewCategoryRepository(db, log)
	categorySvc := service.NewCategoryService(categoryRepo, log, nil)
	productSvc := service.NewProductService(repository.NewProductRepository(db, log, categoryRepo), log, nil)
	inventorySvc := service.NewInventoryService(repository.NewInventoryRepository(db, log), log, nil)

	api := NewAPIHandler(userSvc, log, nil)
	categoryAPI := NewCategoryAPIHandler(categorySvc, log, nil)
	productAPI := NewProductAPIHandler(productSvc, log, nil)
	inventoryAPI := NewInventoryAPIHandler(inventorySvc, log, nil)
	health := NewHealthHandler(log, db, nil)

	r := chi.NewRouter()
	r.Route("/health", func(sr chi.Router) {
		sr.Get("/live", health.Live)
		sr.Get("/ready", health.Ready)
	})
	r.Route("/api/v1", func(ar chi.Router) {
		ar.Post("/users", api.CreateUser)
		ar.Get("/users/{id}", api.GetUser)
		ar.Get("/users", api.ListUsers)
		ar.Route("/categories", func(cr chi.Router) {
			cr.Post("/", categoryAPI.CreateCategory)
			cr.Get("/{id}", categoryAPI.GetCategory)
			cr.Get("/", categoryAPI.ListCategories)
			cr.Put("/{id}", categoryAPI.UpdateCategory)
			cr.Delete("/{id}", categoryAPI.DeleteCategory)
		})
		ar.Route("/products", func(pr chi.Router) {
			pr.Post("/", productAPI.CreateProduct)
			pr.Get("/{id}", productAPI.GetProduct)
			pr.Get("/", productAPI.ListProducts)
			pr.Put("/{id}", productAPI.UpdateProduct)
			pr.Delete("/{id}", productAPI.DeleteProduct)
		})
		ar.Route("/inventories", func(ir chi.Router) {
			ir.Post("/", inventoryAPI.CreateInventory)
			ir.Get("/{id}", inventoryAPI.GetInventory)
			ir.Get("/", inventoryAPI.ListInventories)
			ir.Put("/{id}", inventoryAPI.UpdateInventory)
			ir.Delete("/{id}", inventoryAPI.DeleteInventory)
		})
	})

	return &testApp{router: r, db: db}
}

func migrationsPath(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations")
}

func TestHealthHandlers(t *testing.T) {
	app := newTestApp(t)

	rec := httptest.NewRecorder()
	app.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("live status=%d body=%q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	app.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("ready status=%d body=%s", rec.Code, rec.Body.String())
	}
	var checks map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &checks); err != nil {
		t.Fatalf("decode ready response: %v", err)
	}
	if checks["database"] != "healthy" || checks["self"] != "healthy" {
		t.Fatalf("checks = %#v", checks)
	}

	if err := app.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
	rec = httptest.NewRecorder()
	app.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready after close status=%d, want 503", rec.Code)
	}
}

func TestUserAPIFlowAndErrors(t *testing.T) {
	app := newTestApp(t)

	created := doJSON[repository.User](t, app.router, http.MethodPost, "/api/v1/users", map[string]any{
		"email": "alice@example.com",
		"name":  "Alice",
	}, http.StatusCreated)
	if created.ID == "" {
		t.Fatal("created user id is empty")
	}

	duplicate := doProblem(t, app.router, http.MethodPost, "/api/v1/users", map[string]any{
		"email": "alice@example.com",
		"name":  "Alice Again",
	}, http.StatusConflict)
	if duplicate.Title != "Conflict" {
		t.Fatalf("duplicate title = %q", duplicate.Title)
	}

	got := doJSON[repository.User](t, app.router, http.MethodGet, "/api/v1/users/"+created.ID, nil, http.StatusOK)
	if got.Email != created.Email {
		t.Fatalf("got email = %q", got.Email)
	}

	users := doJSON[[]repository.User](t, app.router, http.MethodGet, "/api/v1/users?page=1&limit=10", nil, http.StatusOK)
	if len(users) != 1 {
		t.Fatalf("users len = %d, want 1", len(users))
	}

	validationProblem := doProblem(t, app.router, http.MethodPost, "/api/v1/users", map[string]any{
		"email": "bad",
		"name":  "",
	}, http.StatusUnprocessableEntity)
	if validationProblem.Title != "Validation Failed" || len(validationProblem.Errors) == 0 {
		t.Fatalf("validation problem = %#v", validationProblem)
	}

	notFound := doProblem(t, app.router, http.MethodGet, "/api/v1/users/550e8400-e29b-41d4-a716-446655440000", nil, http.StatusNotFound)
	if notFound.Title != "Not Found" {
		t.Fatalf("not found title = %q", notFound.Title)
	}
}

func TestInventoryAPIFlowAndErrors(t *testing.T) {
	app := newTestApp(t)

	category := doJSON[repository.Category](t, app.router, http.MethodPost, "/api/v1/categories/", map[string]any{
		"name": "Electronics",
	}, http.StatusCreated)

	missingCategory := doProblem(t, app.router, http.MethodPost, "/api/v1/products/", map[string]any{
		"name":        "Keyboard",
		"description": "Mechanical",
		"price":       120,
		"category_id": "550e8400-e29b-41d4-a716-446655440000",
	}, http.StatusUnprocessableEntity)
	if missingCategory.Detail != "The referenced category does not exist." {
		t.Fatalf("missing category detail = %q", missingCategory.Detail)
	}

	product := doJSON[repository.Product](t, app.router, http.MethodPost, "/api/v1/products/", map[string]any{
		"name":        "Keyboard",
		"description": "Mechanical",
		"price":       120,
		"category_id": category.ID,
	}, http.StatusCreated)
	updatedProduct := doJSON[repository.Product](t, app.router, http.MethodPut, "/api/v1/products/"+product.ID, map[string]any{
		"name":        "Keyboard Pro",
		"description": "Updated",
		"price":       150,
		"category_id": category.ID,
	}, http.StatusOK)
	if updatedProduct.Name != "Keyboard Pro" {
		t.Fatalf("updated product = %#v", updatedProduct)
	}
	products := doJSON[[]repository.Product](t, app.router, http.MethodGet, "/api/v1/products/?page=1&limit=10", nil, http.StatusOK)
	if len(products) != 1 {
		t.Fatalf("products len = %d, want 1", len(products))
	}

	inventory := doJSON[repository.Inventory](t, app.router, http.MethodPost, "/api/v1/inventories/", map[string]any{
		"product_id": product.ID,
		"quantity":   10,
	}, http.StatusCreated)
	updatedInventory := doJSON[repository.Inventory](t, app.router, http.MethodPut, "/api/v1/inventories/"+inventory.ID, map[string]any{
		"product_id": product.ID,
		"quantity":   12,
	}, http.StatusOK)
	if updatedInventory.Quantity != 12 {
		t.Fatalf("updated inventory = %#v", updatedInventory)
	}
	inventories := doJSON[[]repository.Inventory](t, app.router, http.MethodGet, "/api/v1/inventories/?page=1&limit=10", nil, http.StatusOK)
	if len(inventories) != 1 {
		t.Fatalf("inventories len = %d, want 1", len(inventories))
	}

	gotCategory := doJSON[repository.Category](t, app.router, http.MethodGet, "/api/v1/categories/"+category.ID, nil, http.StatusOK)
	if gotCategory.ID != category.ID {
		t.Fatalf("got category = %#v", gotCategory)
	}
	updatedCategory := doJSON[repository.Category](t, app.router, http.MethodPut, "/api/v1/categories/"+category.ID, map[string]any{
		"name": "Computers",
	}, http.StatusOK)
	if updatedCategory.Name != "Computers" {
		t.Fatalf("updated category = %#v", updatedCategory)
	}
	categories := doJSON[[]repository.Category](t, app.router, http.MethodGet, "/api/v1/categories/?page=1&limit=10", nil, http.StatusOK)
	if len(categories) != 1 {
		t.Fatalf("categories len = %d, want 1", len(categories))
	}

	invalidUUID := doProblem(t, app.router, http.MethodGet, "/api/v1/products/not-a-uuid", nil, http.StatusUnprocessableEntity)
	if invalidUUID.Title != "Validation Failed" {
		t.Fatalf("invalid uuid title = %q", invalidUUID.Title)
	}

	doJSON[any](t, app.router, http.MethodDelete, "/api/v1/inventories/"+inventory.ID, nil, http.StatusOK)
	doJSON[any](t, app.router, http.MethodDelete, "/api/v1/products/"+product.ID, nil, http.StatusOK)
	doJSON[any](t, app.router, http.MethodDelete, "/api/v1/categories/"+category.ID, nil, http.StatusOK)

	notFound := doProblem(t, app.router, http.MethodDelete, "/api/v1/products/"+product.ID, nil, http.StatusNotFound)
	if notFound.Title != "Not Found" {
		t.Fatalf("not found title = %q", notFound.Title)
	}
}

func TestAPIValidationAndInternalErrorBranches(t *testing.T) {
	app := newTestApp(t)

	tests := []struct {
		name   string
		method string
		path   string
		body   any
		status int
	}{
		{
			name:   "list users invalid limit",
			method: http.MethodGet,
			path:   "/api/v1/users?page=1&limit=101",
			status: http.StatusUnprocessableEntity,
		},
		{
			name:   "category invalid id",
			method: http.MethodGet,
			path:   "/api/v1/categories/not-a-uuid",
			status: http.StatusUnprocessableEntity,
		},
		{
			name:   "category invalid create body",
			method: http.MethodPost,
			path:   "/api/v1/categories/",
			body:   map[string]any{"name": ""},
			status: http.StatusUnprocessableEntity,
		},
		{
			name:   "category not found",
			method: http.MethodGet,
			path:   "/api/v1/categories/550e8400-e29b-41d4-a716-446655440000",
			status: http.StatusNotFound,
		},
		{
			name:   "list categories invalid limit",
			method: http.MethodGet,
			path:   "/api/v1/categories/?page=1&limit=101",
			status: http.StatusUnprocessableEntity,
		},
		{
			name:   "category update not found",
			method: http.MethodPut,
			path:   "/api/v1/categories/550e8400-e29b-41d4-a716-446655440000",
			body:   map[string]any{"name": "Missing"},
			status: http.StatusNotFound,
		},
		{
			name:   "category delete invalid id",
			method: http.MethodDelete,
			path:   "/api/v1/categories/not-a-uuid",
			status: http.StatusUnprocessableEntity,
		},
		{
			name:   "category delete not found",
			method: http.MethodDelete,
			path:   "/api/v1/categories/550e8400-e29b-41d4-a716-446655440000",
			status: http.StatusNotFound,
		},
		{
			name:   "product invalid create body",
			method: http.MethodPost,
			path:   "/api/v1/products/",
			body:   map[string]any{"name": "", "description": "", "price": -1, "category_id": "not-a-uuid"},
			status: http.StatusUnprocessableEntity,
		},
		{
			name:   "product get not found",
			method: http.MethodGet,
			path:   "/api/v1/products/550e8400-e29b-41d4-a716-446655440000",
			status: http.StatusNotFound,
		},
		{
			name:   "product update not found",
			method: http.MethodPut,
			path:   "/api/v1/products/550e8400-e29b-41d4-a716-446655440000",
			body:   map[string]any{"name": "Missing", "description": "Missing", "price": 1, "category_id": "550e8400-e29b-41d4-a716-446655440001"},
			status: http.StatusNotFound,
		},
		{
			name:   "product invalid update id",
			method: http.MethodPut,
			path:   "/api/v1/products/not-a-uuid",
			body:   map[string]any{},
			status: http.StatusUnprocessableEntity,
		},
		{
			name:   "product invalid delete id",
			method: http.MethodDelete,
			path:   "/api/v1/products/not-a-uuid",
			status: http.StatusUnprocessableEntity,
		},
		{
			name:   "inventory invalid create body",
			method: http.MethodPost,
			path:   "/api/v1/inventories/",
			body:   map[string]any{"product_id": "not-a-uuid", "quantity": -1},
			status: http.StatusUnprocessableEntity,
		},
		{
			name:   "inventory not found",
			method: http.MethodGet,
			path:   "/api/v1/inventories/550e8400-e29b-41d4-a716-446655440000",
			status: http.StatusNotFound,
		},
		{
			name:   "inventory update not found",
			method: http.MethodPut,
			path:   "/api/v1/inventories/550e8400-e29b-41d4-a716-446655440000",
			body:   map[string]any{"product_id": "550e8400-e29b-41d4-a716-446655440001", "quantity": 1},
			status: http.StatusNotFound,
		},
		{
			name:   "inventory update invalid id",
			method: http.MethodPut,
			path:   "/api/v1/inventories/not-a-uuid",
			body:   map[string]any{"product_id": "550e8400-e29b-41d4-a716-446655440001", "quantity": 1},
			status: http.StatusUnprocessableEntity,
		},
		{
			name:   "inventory invalid delete id",
			method: http.MethodDelete,
			path:   "/api/v1/inventories/not-a-uuid",
			status: http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problem := doProblem(t, app.router, tt.method, tt.path, tt.body, tt.status)
			if problem.Title == "" {
				t.Fatalf("problem title is empty: %#v", problem)
			}
		})
	}

	closedApp := newTestApp(t)
	if err := closedApp.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
	internalTests := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{
			name:   "create user internal error",
			method: http.MethodPost,
			path:   "/api/v1/users",
			body:   map[string]any{"email": "closed@example.com", "name": "Closed"},
		},
		{
			name:   "list users internal error",
			method: http.MethodGet,
			path:   "/api/v1/users",
		},
		{
			name:   "create category internal error",
			method: http.MethodPost,
			path:   "/api/v1/categories/",
			body:   map[string]any{"name": "Closed"},
		},
		{
			name:   "update category internal error",
			method: http.MethodPut,
			path:   "/api/v1/categories/550e8400-e29b-41d4-a716-446655440000",
			body:   map[string]any{"name": "Closed"},
		},
		{
			name:   "create product internal error",
			method: http.MethodPost,
			path:   "/api/v1/products/",
			body:   map[string]any{"name": "Closed", "description": "Closed", "price": 1, "category_id": "550e8400-e29b-41d4-a716-446655440000"},
		},
		{
			name:   "list products internal error",
			method: http.MethodGet,
			path:   "/api/v1/products/",
		},
		{
			name:   "create inventory internal error",
			method: http.MethodPost,
			path:   "/api/v1/inventories/",
			body:   map[string]any{"product_id": "550e8400-e29b-41d4-a716-446655440000", "quantity": 1},
		},
		{
			name:   "list inventories internal error",
			method: http.MethodGet,
			path:   "/api/v1/inventories/",
		},
	}
	for _, tt := range internalTests {
		t.Run(tt.name, func(t *testing.T) {
			problem := doProblem(t, closedApp.router, tt.method, tt.path, tt.body, http.StatusInternalServerError)
			if problem.Title != "Internal Server Error" {
				t.Fatalf("problem = %#v", problem)
			}
		})
	}
}

func doJSON[T any](t *testing.T, handler http.Handler, method, path string, payload any, wantStatus int) T {
	t.Helper()

	rec := performRequest(t, handler, method, path, payload)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	var env struct {
		Data T `json:"data"`
		Meta struct {
			Pagination *apiresponse.PaginationMeta `json:"pagination,omitempty"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v; body=%s", err, rec.Body.String())
	}
	return env.Data
}

func doProblem(t *testing.T, handler http.Handler, method, path string, payload any, wantStatus int) apiresponse.Problem {
	t.Helper()

	rec := performRequest(t, handler, method, path, payload)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	var problem apiresponse.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v; body=%s", err, rec.Body.String())
	}
	return problem
}

func performRequest(t *testing.T, handler http.Handler, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("encode payload: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

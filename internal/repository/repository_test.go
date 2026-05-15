package repository

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/kaungmyathan18/golang-inventory-app/internal/config"
	"github.com/kaungmyathan18/golang-inventory-app/internal/database"
	"go.uber.org/zap"
)

func newTestDB(t *testing.T) *database.DB {
	t.Helper()

	ctx := context.Background()
	db, err := database.New(ctx, config.DatabaseConfig{
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

	if err := database.RunMigrations(ctx, db, migrationsPath(t)); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return db
}

func migrationsPath(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "migrations")
}

func TestParseCreatedAt(t *testing.T) {
	tests := []string{
		"2026-05-15T12:00:00.123456789Z",
		"2026-05-15 12:00:00.123456789 +0000 UTC",
		"2026-05-15 12:00:00.123456789Z",
		"2026-05-15 12:00:00Z",
	}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			got, err := parseCreatedAt(value)
			if err != nil {
				t.Fatalf("parseCreatedAt returned error: %v", err)
			}
			if got.Location() != time.UTC {
				t.Fatalf("location = %v, want UTC", got.Location())
			}
		})
	}

	if _, err := parseCreatedAt("not-a-date"); err == nil {
		t.Fatal("parseCreatedAt returned nil error for invalid date")
	}
}

func TestUserRepositoryCreateGetListAndDuplicate(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db, zap.NewNop())
	ctx := context.Background()

	first, err := repo.Create(ctx, "alice@example.com", "Alice")
	if err != nil {
		t.Fatalf("create first user: %v", err)
	}
	second, err := repo.Create(ctx, "bob@example.com", "Bob")
	if err != nil {
		t.Fatalf("create second user: %v", err)
	}

	got, err := repo.Get(ctx, first.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.Email != first.Email || got.Name != first.Name {
		t.Fatalf("got user = %#v, want %#v", got, first)
	}

	users, total, err := repo.ListPaged(ctx, 0, 10)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if total != 2 || len(users) != 2 {
		t.Fatalf("total=%d len=%d, want 2 users", total, len(users))
	}
	if users[0].ID == "" || users[1].ID == "" || second.ID == "" {
		t.Fatalf("users missing ids: %#v %#v %#v", users, first, second)
	}

	if _, err := repo.Create(ctx, "alice@example.com", "Alice Again"); !errors.Is(err, ErrDuplicateEmail) {
		t.Fatalf("duplicate create error = %v, want ErrDuplicateEmail", err)
	}
	if _, err := repo.Get(ctx, "550e8400-e29b-41d4-a716-446655440000"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing get error = %v, want ErrNotFound", err)
	}
}

func TestCategoryProductInventoryRepositories(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	log := zap.NewNop()
	categoryRepo := NewCategoryRepository(db, log)
	productRepo := NewProductRepository(db, log, categoryRepo)
	inventoryRepo := NewInventoryRepository(db, log)

	category, err := categoryRepo.Create(ctx, "Electronics")
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	updatedCategory, err := categoryRepo.Update(ctx, category.ID, "Computers")
	if err != nil {
		t.Fatalf("update category: %v", err)
	}
	if updatedCategory.Name != "Computers" {
		t.Fatalf("updated category name = %q", updatedCategory.Name)
	}

	if _, err := productRepo.Create(ctx, "Keyboard", "Mechanical keyboard", 129.99, "550e8400-e29b-41d4-a716-446655440000"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing category error = %v, want ErrNotFound", err)
	}
	product, err := productRepo.Create(ctx, "Keyboard", "Mechanical keyboard", 129.99, category.ID)
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	gotProduct, err := productRepo.Get(ctx, product.ID)
	if err != nil {
		t.Fatalf("get product: %v", err)
	}
	if gotProduct.Name != product.Name || gotProduct.CategoryID != category.ID {
		t.Fatalf("got product = %#v", gotProduct)
	}
	updatedProduct, err := productRepo.Update(ctx, product.ID, "Keyboard Pro", "Updated", 149.99, category.ID)
	if err != nil {
		t.Fatalf("update product: %v", err)
	}
	if updatedProduct.Name != "Keyboard Pro" || updatedProduct.Price != 149.99 {
		t.Fatalf("updated product = %#v", updatedProduct)
	}

	inventory, err := inventoryRepo.Create(ctx, product.ID, 20)
	if err != nil {
		t.Fatalf("create inventory: %v", err)
	}
	gotInventory, err := inventoryRepo.Get(ctx, inventory.ID)
	if err != nil {
		t.Fatalf("get inventory: %v", err)
	}
	if gotInventory.ProductID != product.ID || gotInventory.Quantity != 20 {
		t.Fatalf("got inventory = %#v", gotInventory)
	}
	updatedInventory, err := inventoryRepo.Update(ctx, inventory.ID, product.ID, 15)
	if err != nil {
		t.Fatalf("update inventory: %v", err)
	}
	if updatedInventory.Quantity != 15 {
		t.Fatalf("updated inventory = %#v", updatedInventory)
	}

	categories, categoryTotal, err := categoryRepo.ListPaged(ctx, 0, 10)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if categoryTotal != 1 || len(categories) != 1 {
		t.Fatalf("category total=%d len=%d", categoryTotal, len(categories))
	}
	products, productTotal, err := productRepo.ListPaged(ctx, 0, 10)
	if err != nil {
		t.Fatalf("list products: %v", err)
	}
	if productTotal != 1 || len(products) != 1 {
		t.Fatalf("product total=%d len=%d", productTotal, len(products))
	}
	inventories, inventoryTotal, err := inventoryRepo.ListPaged(ctx, 0, 10)
	if err != nil {
		t.Fatalf("list inventories: %v", err)
	}
	if inventoryTotal != 1 || len(inventories) != 1 {
		t.Fatalf("inventory total=%d len=%d", inventoryTotal, len(inventories))
	}

	if _, err := categoryRepo.Update(ctx, "550e8400-e29b-41d4-a716-446655440000", "Missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing category update error = %v", err)
	}
	if _, err := productRepo.Update(ctx, "550e8400-e29b-41d4-a716-446655440000", "Missing", "Missing", 1, category.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing product update error = %v", err)
	}
	if _, err := inventoryRepo.Update(ctx, "550e8400-e29b-41d4-a716-446655440000", product.ID, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing inventory update error = %v", err)
	}

	if err := inventoryRepo.Delete(ctx, inventory.ID); err != nil {
		t.Fatalf("delete inventory: %v", err)
	}
	if err := productRepo.Delete(ctx, product.ID); err != nil {
		t.Fatalf("delete product: %v", err)
	}
	if err := categoryRepo.Delete(ctx, category.ID); err != nil {
		t.Fatalf("delete category: %v", err)
	}
	if err := inventoryRepo.Delete(ctx, inventory.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing inventory delete error = %v", err)
	}
	if err := productRepo.Delete(ctx, product.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing product delete error = %v", err)
	}
	if err := categoryRepo.Delete(ctx, category.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing category delete error = %v", err)
	}
}

func TestRepositoryErrorBranches(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	log := zap.NewNop()
	userRepo := NewUserRepository(db, log)
	categoryRepo := NewCategoryRepository(db, log)
	productRepo := NewProductRepository(db, log, categoryRepo)
	inventoryRepo := NewInventoryRepository(db, log)

	if _, err := categoryRepo.Get(ctx, "550e8400-e29b-41d4-a716-446655440000"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing category get error = %v", err)
	}
	if _, err := productRepo.Get(ctx, "550e8400-e29b-41d4-a716-446655440000"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing product get error = %v", err)
	}
	if _, err := inventoryRepo.Get(ctx, "550e8400-e29b-41d4-a716-446655440000"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing inventory get error = %v", err)
	}

	productRepoWithoutCategoryCheck := NewProductRepository(db, log, nil)
	looseProduct, err := productRepoWithoutCategoryCheck.Create(ctx, "Loose", "No category check", 1, "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("product create without category check: %v", err)
	}
	if looseProduct.ID == "" {
		t.Fatal("loose product id is empty")
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	if _, err := userRepo.Create(ctx, "closed@example.com", "Closed"); err == nil {
		t.Fatal("user create on closed db returned nil error")
	}
	if _, err := userRepo.Get(ctx, "550e8400-e29b-41d4-a716-446655440000"); err == nil {
		t.Fatal("user get on closed db returned nil error")
	}
	if _, _, err := userRepo.ListPaged(ctx, 0, 10); err == nil {
		t.Fatal("user list on closed db returned nil error")
	}
	if _, _, err := categoryRepo.ListPaged(ctx, 0, 10); err == nil {
		t.Fatal("category list on closed db returned nil error")
	}
	if _, err := categoryRepo.Create(ctx, "Closed"); err == nil {
		t.Fatal("category create on closed db returned nil error")
	}
	if _, err := productRepo.Create(ctx, "Closed", "Closed", 1, "550e8400-e29b-41d4-a716-446655440000"); err == nil {
		t.Fatal("product create on closed db returned nil error")
	}
	if _, _, err := productRepo.ListPaged(ctx, 0, 10); err == nil {
		t.Fatal("product list on closed db returned nil error")
	}
	if _, _, err := inventoryRepo.ListPaged(ctx, 0, 10); err == nil {
		t.Fatal("inventory list on closed db returned nil error")
	}
}

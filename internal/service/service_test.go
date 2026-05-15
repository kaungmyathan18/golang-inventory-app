package service

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/kaungmyathan18/golang-inventory-app/internal/config"
	"github.com/kaungmyathan18/golang-inventory-app/internal/database"
	"github.com/kaungmyathan18/golang-inventory-app/internal/repository"
	"go.uber.org/zap"
)

func newServiceTestDB(t *testing.T) *database.DB {
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

func TestUserService(t *testing.T) {
	db := newServiceTestDB(t)
	svc := NewUserService(repository.NewUserRepository(db, zap.NewNop()), zap.NewNop(), nil)
	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "alice@example.com", "Alice")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if _, err := svc.CreateUser(ctx, "alice@example.com", "Alice Again"); !errors.Is(err, repository.ErrDuplicateEmail) {
		t.Fatalf("duplicate error = %v, want ErrDuplicateEmail", err)
	}

	got, err := svc.GetUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.Email != "alice@example.com" {
		t.Fatalf("email = %q", got.Email)
	}

	users, total, err := svc.ListUsersPaged(ctx, 0, 0)
	if err != nil {
		t.Fatalf("ListUsersPaged: %v", err)
	}
	if total != 1 || len(users) != 1 {
		t.Fatalf("total=%d len=%d, want one user", total, len(users))
	}
	if err := svc.HandleQueuePayload(ctx, "payload"); err != nil {
		t.Fatalf("HandleQueuePayload: %v", err)
	}
	if err := svc.PingDeps(ctx); err != nil {
		t.Fatalf("PingDeps: %v", err)
	}
}

func TestProductCategoryInventoryServices(t *testing.T) {
	db := newServiceTestDB(t)
	log := zap.NewNop()
	ctx := context.Background()

	categoryRepo := repository.NewCategoryRepository(db, log)
	categorySvc := NewCategoryService(categoryRepo, log, nil)
	productRepo := repository.NewProductRepository(db, log, categoryRepo)
	productSvc := NewProductService(productRepo, log, nil)
	inventorySvc := NewInventoryService(repository.NewInventoryRepository(db, log), log, nil)

	category, err := categorySvc.CreateCategory(ctx, "Electronics")
	if err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}
	if _, err := categorySvc.UpdateCategory(ctx, category.ID, "Computers"); err != nil {
		t.Fatalf("UpdateCategory: %v", err)
	}
	if _, err := categorySvc.GetCategory(ctx, category.ID); err != nil {
		t.Fatalf("GetCategory: %v", err)
	}
	if _, _, err := categorySvc.ListCategoriesPaged(ctx, 0, 0); err != nil {
		t.Fatalf("ListCategoriesPaged: %v", err)
	}
	if err := categorySvc.PingDeps(ctx); err != nil {
		t.Fatalf("Category PingDeps: %v", err)
	}

	product, err := productSvc.CreateProduct(ctx, "Keyboard", "Mechanical", 100, category.ID)
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	if _, err := productSvc.GetProduct(ctx, product.ID); err != nil {
		t.Fatalf("GetProduct: %v", err)
	}
	if _, _, err := productSvc.ListProductsPaged(ctx, 0, 0); err != nil {
		t.Fatalf("ListProductsPaged: %v", err)
	}
	if _, err := productSvc.UpdateProduct(ctx, product.ID, "Keyboard Pro", "Updated", 150, category.ID); err != nil {
		t.Fatalf("UpdateProduct: %v", err)
	}
	if err := productSvc.PingDeps(ctx); err != nil {
		t.Fatalf("Product PingDeps: %v", err)
	}

	inventory, err := inventorySvc.CreateInventory(ctx, product.ID, 5)
	if err != nil {
		t.Fatalf("CreateInventory: %v", err)
	}
	if _, err := inventorySvc.GetInventory(ctx, inventory.ID); err != nil {
		t.Fatalf("GetInventory: %v", err)
	}
	if _, _, err := inventorySvc.ListInventoriesPaged(ctx, 0, 0); err != nil {
		t.Fatalf("ListInventoriesPaged: %v", err)
	}
	if _, err := inventorySvc.UpdateInventory(ctx, inventory.ID, product.ID, 7); err != nil {
		t.Fatalf("UpdateInventory: %v", err)
	}
	if err := inventorySvc.PingDeps(ctx); err != nil {
		t.Fatalf("Inventory PingDeps: %v", err)
	}

	if err := inventorySvc.DeleteInventory(ctx, inventory.ID); err != nil {
		t.Fatalf("DeleteInventory: %v", err)
	}
	if err := productSvc.DeleteProduct(ctx, product.ID); err != nil {
		t.Fatalf("DeleteProduct: %v", err)
	}
	if err := categorySvc.DeleteCategory(ctx, category.ID); err != nil {
		t.Fatalf("DeleteCategory: %v", err)
	}
}

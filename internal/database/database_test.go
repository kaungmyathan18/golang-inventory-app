package database

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kaungmyathan18/golang-inventory-app/internal/config"
)

func TestNewSQLiteAndRunMigrations(t *testing.T) {
	dir := t.TempDir()
	migrationsDir := filepath.Join(dir, "migrations")
	if err := os.Mkdir(migrationsDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, "002_second.sql"), []byte(`INSERT INTO items (id, name) VALUES (1, 'first');`), 0o644); err != nil {
		t.Fatalf("write migration: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, "001_first.sql"), []byte(`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT NOT NULL);`), 0o644); err != nil {
		t.Fatalf("write migration: %v", err)
	}

	db, err := New(context.Background(), config.DatabaseConfig{
		Driver:          "sqlite",
		DSN:             "file:" + filepath.Join(dir, "nested", "test.db") + "?_foreign_keys=on",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatalf("New sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
	})

	if err := RunMigrations(context.Background(), db, migrationsDir); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	var name string
	if err := db.SQL.QueryRowContext(context.Background(), `SELECT name FROM items WHERE id = 1`).Scan(&name); err != nil {
		t.Fatalf("query migrated data: %v", err)
	}
	if name != "first" {
		t.Fatalf("name = %q, want first", name)
	}
	if err := db.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestNewRejectsUnsupportedDriver(t *testing.T) {
	_, err := New(context.Background(), config.DatabaseConfig{Driver: "postgres"})
	if err == nil || !strings.Contains(err.Error(), "unsupported driver") {
		t.Fatalf("error = %v, want unsupported driver", err)
	}

	_, err = New(context.Background(), config.DatabaseConfig{Driver: "mysql"})
	if err == nil || !strings.Contains(err.Error(), "unknown driver") {
		t.Fatalf("mysql error = %v, want unknown driver", err)
	}
}

func TestEnsureSQLiteDir(t *testing.T) {
	if err := ensureSQLiteDir(":memory:"); err != nil {
		t.Fatalf("memory dir: %v", err)
	}
	if err := ensureSQLiteDir("file::memory:?cache=shared"); err != nil {
		t.Fatalf("shared memory dir: %v", err)
	}
	path := filepath.Join(t.TempDir(), "nested", "app.db")
	if err := ensureSQLiteDir("file:" + path + "?_foreign_keys=on"); err != nil {
		t.Fatalf("file dir: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("created dir stat: %v", err)
	}
}

func TestRunMigrationsErrors(t *testing.T) {
	db, err := New(context.Background(), config.DatabaseConfig{
		Driver:          "sqlite",
		DSN:             ":memory:",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatalf("New sqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
	})

	if err := RunMigrations(context.Background(), db, filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("RunMigrations missing dir returned nil error")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "001_bad.sql"), []byte(`CREATE TABLE broken (`), 0o644); err != nil {
		t.Fatalf("write bad migration: %v", err)
	}
	if err := RunMigrations(context.Background(), db, dir); err == nil || !strings.Contains(err.Error(), "001_bad.sql") {
		t.Fatalf("bad migration error = %v", err)
	}
}

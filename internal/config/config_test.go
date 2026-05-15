package config

import (
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestDefaultConfigReadsEnvironment(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	t.Setenv("APP_ENV", "test")
	t.Setenv("DB_DSN", "file:test.db")
	t.Setenv("DB_AUTO_MIGRATE", "true")
	t.Setenv("DB_PORT", "3306")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("OTEL_ENABLED", "false")
	t.Setenv("LOG_LEVEL", "debug")
	if err := InitEnv(); err != nil {
		t.Fatalf("InitEnv: %v", err)
	}

	cfg := DefaultConfig("inventory")

	if cfg.App.Name != "inventory" || cfg.App.Environment != "test" {
		t.Fatalf("app config = %#v", cfg.App)
	}
	if cfg.Database.DSN != "file:test.db" || !cfg.Database.AutoMigrate || cfg.Database.Port != 3306 {
		t.Fatalf("database config = %#v", cfg.Database)
	}
	if cfg.Cache.Addr != "redis:6379" {
		t.Fatalf("cache addr = %q", cfg.Cache.Addr)
	}
	if cfg.Otel.Enabled {
		t.Fatal("otel enabled = true, want false")
	}
	if cfg.Log.Level != "debug" {
		t.Fatalf("log level = %q", cfg.Log.Level)
	}
}

func TestDatabaseDSNs(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		Name:     "db",
		SSLMode:  "disable",
	}
	if got := cfg.PostgresDSN(); got != "host=localhost port=5432 user=user password=pass dbname=db sslmode=disable" {
		t.Fatalf("PostgresDSN = %q", got)
	}
	cfg.Port = 3306
	if got := cfg.MySQLDSN(); got != "user:pass@tcp(localhost:3306)/db" {
		t.Fatalf("MySQLDSN = %q", got)
	}
}

func TestDefaultConfigDefaults(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	cfg := DefaultConfig("inventory")

	if cfg.Server.Port != 8080 {
		t.Fatalf("server port = %d", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 15*time.Second {
		t.Fatalf("read timeout = %s", cfg.Server.ReadTimeout)
	}
	if !cfg.Server.CORS.Enabled || len(cfg.Server.CORS.AllowedMethods) == 0 {
		t.Fatalf("cors config = %#v", cfg.Server.CORS)
	}
	if cfg.Database.Driver != "sqlite" || cfg.Database.MigrationsPath != "./migrations" {
		t.Fatalf("database defaults = %#v", cfg.Database)
	}
}

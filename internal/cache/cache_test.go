package cache

import (
	"context"
	"testing"

	"github.com/kaungmyathan18/golang-inventory-app/internal/config"
	"github.com/redis/go-redis/v9"
)

func TestNewReturnsPingError(t *testing.T) {
	if _, err := New(config.CacheConfig{Addr: "127.0.0.1:0"}); err == nil {
		t.Fatal("New returned nil error for unreachable Redis")
	}
}

func TestClientPingAndClose(t *testing.T) {
	client := &Client{rdb: redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})}

	if err := client.Ping(context.Background()); err == nil {
		t.Fatal("Ping returned nil error for unreachable Redis")
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

package database_test

import (
	"context"
	"os"
	"testing"
	"time"

	"kurbantrack-backend/database"
)

func TestPostgresConnection_Live(t *testing.T) {
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		t.Skip("Melewati live database test: environment variable TEST_DATABASE_URL tidak disetel")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, testDBURL)
	if err != nil {
		t.Fatalf("Gagal membuat connection pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Gagal melakukan ping ke PostgreSQL pada TEST_DATABASE_URL: %v", err)
	}
}

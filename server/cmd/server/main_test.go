package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunRejectsProductionWithoutJWTSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DATABASE_URL", "")

	err := run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("run() error = %v, want missing JWT_SECRET error", err)
	}
}

func TestRunDevelopmentWithoutDependenciesStopsWithContext(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("SERVER_ADDR", "127.0.0.1:0")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_ADDR", "127.0.0.1:1")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("JWT_SECRET", "dev-test-secret")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := run(ctx); err != nil {
		t.Fatalf("run() error = %v, want clean context shutdown", err)
	}
}

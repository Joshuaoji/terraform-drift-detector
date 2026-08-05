package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/terraform-drift-detector/driftdetect/internal/store/postgres"
	"github.com/terraform-drift-detector/driftdetect/internal/store/sqlite"
)

// Closer is implemented by stores that need explicit shutdown.
type Closer interface {
	Close() error
}

// Open connects to SQLite (file path) or PostgreSQL (postgres:// DSN).
func Open(dsn string) (Store, error) {
	if isPostgresDSN(dsn) {
		return postgres.Open(dsn)
	}
	return sqlite.Open(dsn)
}

func isPostgresDSN(dsn string) bool {
	return strings.HasPrefix(dsn, "postgres://") ||
		strings.HasPrefix(dsn, "postgresql://")
}

// Ping checks database connectivity.
func Ping(ctx context.Context, st Store) error {
	if p, ok := st.(interface{ Ping(context.Context) error }); ok {
		return p.Ping(ctx)
	}
	return nil
}

// Close closes the store if it supports Close.
func Close(st Store) error {
	if c, ok := st.(Closer); ok {
		return c.Close()
	}
	return nil
}

// BackendName returns a human-readable store backend name.
func BackendName(dsn string) string {
	if isPostgresDSN(dsn) {
		return "postgres"
	}
	return "sqlite"
}

// ValidateDSN returns an error for empty DSN.
func ValidateDSN(dsn string) error {
	if strings.TrimSpace(dsn) == "" {
		return fmt.Errorf("database DSN is required")
	}
	return nil
}

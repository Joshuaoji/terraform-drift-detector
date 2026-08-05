package store_test

import (
	"testing"

	"github.com/terraform-drift-detector/driftdetect/internal/store"
)

func TestBackendName(t *testing.T) {
	if got := store.BackendName("driftdetect.db"); got != "sqlite" {
		t.Fatalf("BackendName sqlite: got %q", got)
	}
	if got := store.BackendName("postgres://localhost/driftdetect"); got != "postgres" {
		t.Fatalf("BackendName postgres: got %q", got)
	}
}

func TestOpenSQLite(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(st)
}

func TestValidateDSN(t *testing.T) {
	if err := store.ValidateDSN(""); err == nil {
		t.Fatal("expected error for empty DSN")
	}
	if err := store.ValidateDSN("driftdetect.db"); err != nil {
		t.Fatal(err)
	}
}

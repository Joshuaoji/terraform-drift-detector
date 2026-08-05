package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/terraform-drift-detector/driftdetect/internal/api"
	"github.com/terraform-drift-detector/driftdetect/internal/scan"
	"github.com/terraform-drift-detector/driftdetect/internal/store/sqlite"
)

func TestWebHandler_ServesIndex(t *testing.T) {
	st, err := sqlite.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	scanner := scan.NewScanner()
	svc := scan.NewService(scanner, st)
	server := api.NewServer(svc, st, nil, nil, api.ServerOptions{DBBackend: "sqlite"})

	tests := []struct {
		path       string
		wantStatus int
	}{
		{"/", http.StatusOK},
		{"/index.html", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			server.Handler().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("GET %s: status %d, want %d, body: %q", tt.path, rec.Code, tt.wantStatus, rec.Body.String())
			}
			if !strings.Contains(strings.ToLower(rec.Body.String()), "<!doctype html>") {
				t.Fatalf("GET %s: expected HTML body", tt.path)
			}
		})
	}
}

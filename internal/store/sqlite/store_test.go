package sqlite_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/terraform-drift-detector/driftdetect/internal/store/sqlite"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

func TestStore_CreateAndGetScan(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	defer os.Remove(dbPath)

	opts := models.ScanOptions{
		StateSource: models.StateSource{Backend: "local", Path: "/tmp/state.tfstate"},
		Provider:    models.ProviderAWS,
	}
	record, err := st.CreateScan(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != models.ScanPending {
		t.Fatalf("expected pending status, got %s", record.Status)
	}

	if err := st.UpdateStatus(context.Background(), record.ID, models.ScanRunning, ""); err != nil {
		t.Fatal(err)
	}

	report := models.DriftReport{
		ScanID:      record.ID,
		StartedAt:   time.Now().UTC(),
		CompletedAt: time.Now().UTC(),
		StateSource: "/tmp/state.tfstate",
		Provider:    models.ProviderAWS,
		Summary:     models.DriftSummary{TotalDrifts: 0},
	}
	if err := st.SaveReport(context.Background(), report); err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateStatus(context.Background(), record.ID, models.ScanCompleted, ""); err != nil {
		t.Fatal(err)
	}

	got, err := st.GetScan(context.Background(), record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != models.ScanCompleted {
		t.Fatalf("expected completed, got %s", got.Status)
	}
	if got.Report == nil {
		t.Fatal("expected report to be stored")
	}
}

func TestStore_ListScanSummaries(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	opts := models.ScanOptions{
		StateSource: models.StateSource{Backend: "local", Path: "/tmp/state.tfstate"},
		Provider:    models.ProviderAWS,
		ProfileName: "prod-aws",
	}
	record, err := st.CreateScan(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}

	report := models.DriftReport{
		ScanID:      record.ID,
		StartedAt:   time.Now().UTC(),
		CompletedAt: time.Now().UTC(),
		StateSource: "/tmp/state.tfstate",
		Provider:    models.ProviderAWS,
		Summary:     models.DriftSummary{TotalDrifts: 3},
	}
	if err := st.SaveReport(context.Background(), report); err != nil {
		t.Fatal(err)
	}
	_ = st.UpdateStatus(context.Background(), record.ID, models.ScanCompleted, "")

	summaries, err := st.ListScanSummaries(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].TotalDrifts != 3 {
		t.Fatalf("expected 3 drifts, got %d", summaries[0].TotalDrifts)
	}
	if summaries[0].ProfileName != "prod-aws" {
		t.Fatalf("expected profile name prod-aws, got %s", summaries[0].ProfileName)
	}
}

func TestStore_ListScans(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	opts := models.ScanOptions{
		StateSource: models.StateSource{Backend: "local", Path: "/tmp/state.tfstate"},
		Provider:    models.ProviderAWS,
	}
	if _, err := st.CreateScan(context.Background(), opts); err != nil {
		t.Fatal(err)
	}

	records, err := st.ListScans(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 scan, got %d", len(records))
	}
}

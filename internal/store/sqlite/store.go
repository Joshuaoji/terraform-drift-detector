package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
	_ "modernc.org/sqlite"
)

// Store is a SQLite-backed scan store.
type Store struct {
	db *sql.DB
}

// Open opens or creates a SQLite database at the given path.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS scans (
    id TEXT PRIMARY KEY,
    status TEXT NOT NULL,
    provider TEXT NOT NULL,
    state_source TEXT NOT NULL,
    options_json TEXT NOT NULL,
    report_json TEXT,
    error TEXT,
    created_at TEXT NOT NULL,
    started_at TEXT,
    completed_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_scans_created_at ON scans(created_at DESC);
`
	_, err := s.db.Exec(schema)
	return err
}

// CreateScan inserts a new pending scan record.
func (s *Store) CreateScan(ctx context.Context, opts models.ScanOptions) (*models.ScanRecord, error) {
	id := uuid.New().String()
	optsJSON, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	stateSource := opts.ResolvedStateSource().Display()
	now := time.Now().UTC()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO scans (id, status, provider, state_source, options_json, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, models.ScanPending, opts.Provider, stateSource, string(optsJSON), now.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("insert scan: %w", err)
	}
	return &models.ScanRecord{
		ID:          id,
		Status:      models.ScanPending,
		Provider:    opts.Provider,
		StateSource: stateSource,
		CreatedAt:   now,
		Options:     opts,
	}, nil
}

// UpdateStatus updates scan lifecycle status.
func (s *Store) UpdateStatus(ctx context.Context, id string, status models.ScanStatus, errMsg string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	switch status {
	case models.ScanRunning:
		_, err := s.db.ExecContext(ctx,
			`UPDATE scans SET status = ?, started_at = ?, error = '' WHERE id = ?`,
			status, now, id,
		)
		return err
	case models.ScanCompleted, models.ScanFailed:
		_, err := s.db.ExecContext(ctx,
			`UPDATE scans SET status = ?, completed_at = ?, error = ? WHERE id = ?`,
			status, now, errMsg, id,
		)
		return err
	default:
		_, err := s.db.ExecContext(ctx, `UPDATE scans SET status = ? WHERE id = ?`, status, id)
		return err
	}
}

// SaveReport persists the drift report for a scan.
func (s *Store) SaveReport(ctx context.Context, report models.DriftReport) error {
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE scans SET report_json = ? WHERE id = ?`, string(data), report.ScanID)
	return err
}

// GetScan retrieves a scan by ID.
func (s *Store) GetScan(ctx context.Context, id string) (*models.ScanRecord, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, status, provider, state_source, options_json, report_json, error, created_at, started_at, completed_at FROM scans WHERE id = ?`,
		id,
	)
	return scanRow(row)
}

// ListScans returns recent scans.
func (s *Store) ListScans(ctx context.Context, limit int) ([]models.ScanRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, status, provider, state_source, options_json, report_json, error, created_at, started_at, completed_at FROM scans ORDER BY created_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.ScanRecord
	for rows.Next() {
		rec, err := scanRows(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, *rec)
	}
	return records, rows.Err()
}

// ListScanSummaries returns recent scans without full report payloads.
func (s *Store) ListScanSummaries(ctx context.Context, limit int) ([]models.ScanSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, status, provider, state_source, options_json, error, created_at, started_at, completed_at,
       COALESCE(CAST(json_extract(report_json, '$.summary.total_drifts') AS INTEGER), 0)
FROM scans ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []models.ScanSummary
	for rows.Next() {
		var summary models.ScanSummary
		var optsJSON sql.NullString
		var errMsg sql.NullString
		var startedAt, completedAt sql.NullString
		var createdAt string

		if err := rows.Scan(&summary.ID, &summary.Status, &summary.Provider, &summary.StateSource,
			&optsJSON, &errMsg, &createdAt, &startedAt, &completedAt, &summary.TotalDrifts); err != nil {
			return nil, err
		}
		summary.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if startedAt.Valid {
			t, _ := time.Parse(time.RFC3339, startedAt.String)
			summary.StartedAt = &t
		}
		if completedAt.Valid {
			t, _ := time.Parse(time.RFC3339, completedAt.String)
			summary.CompletedAt = &t
		}
		if errMsg.Valid {
			summary.Error = errMsg.String
		}
		if optsJSON.Valid {
			var opts models.ScanOptions
			if err := json.Unmarshal([]byte(optsJSON.String), &opts); err == nil {
				summary.ProfileName = opts.ProfileName
			}
		}
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}

func scanRow(row *sql.Row) (*models.ScanRecord, error) {
	var rec models.ScanRecord
	var optsJSON, reportJSON sql.NullString
	var errMsg sql.NullString
	var startedAt, completedAt sql.NullString
	var createdAt string

	err := row.Scan(&rec.ID, &rec.Status, &rec.Provider, &rec.StateSource, &optsJSON, &reportJSON, &errMsg, &createdAt, &startedAt, &completedAt)
	if err != nil {
		return nil, err
	}
	rec.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339, startedAt.String)
		rec.StartedAt = &t
	}
	if completedAt.Valid {
		t, _ := time.Parse(time.RFC3339, completedAt.String)
		rec.CompletedAt = &t
	}
	if errMsg.Valid {
		rec.Error = errMsg.String
	}
	if optsJSON.Valid {
		_ = json.Unmarshal([]byte(optsJSON.String), &rec.Options)
	}
	if reportJSON.Valid && reportJSON.String != "" {
		var report models.DriftReport
		if err := json.Unmarshal([]byte(reportJSON.String), &report); err == nil {
			rec.Report = &report
		}
	}
	return &rec, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRows(rows *sql.Rows) (*models.ScanRecord, error) {
	var rec models.ScanRecord
	var optsJSON, reportJSON sql.NullString
	var errMsg sql.NullString
	var startedAt, completedAt sql.NullString
	var createdAt string

	err := rows.Scan(&rec.ID, &rec.Status, &rec.Provider, &rec.StateSource, &optsJSON, &reportJSON, &errMsg, &createdAt, &startedAt, &completedAt)
	if err != nil {
		return nil, err
	}
	rec.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339, startedAt.String)
		rec.StartedAt = &t
	}
	if completedAt.Valid {
		t, _ := time.Parse(time.RFC3339, completedAt.String)
		rec.CompletedAt = &t
	}
	if errMsg.Valid {
		rec.Error = errMsg.String
	}
	if optsJSON.Valid {
		_ = json.Unmarshal([]byte(optsJSON.String), &rec.Options)
	}
	if reportJSON.Valid && reportJSON.String != "" {
		var report models.DriftReport
		if err := json.Unmarshal([]byte(reportJSON.String), &report); err == nil {
			rec.Report = &report
		}
	}
	return &rec, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

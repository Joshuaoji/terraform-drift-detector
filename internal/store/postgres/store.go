package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Store is a PostgreSQL-backed scan store.
type Store struct {
	db *sql.DB
}

// Open opens a PostgreSQL connection using a DSN.
func Open(dsn string) (*Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Ping verifies database connectivity.
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS scans (
    id TEXT PRIMARY KEY,
    status TEXT NOT NULL,
    provider TEXT NOT NULL,
    state_source TEXT NOT NULL,
    options_json JSONB NOT NULL,
    report_json JSONB,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
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
		`INSERT INTO scans (id, status, provider, state_source, options_json, created_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, models.ScanPending, opts.Provider, stateSource, optsJSON, now,
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
	now := time.Now().UTC()
	switch status {
	case models.ScanRunning:
		_, err := s.db.ExecContext(ctx,
			`UPDATE scans SET status = $1, started_at = $2, error = '' WHERE id = $3`,
			status, now, id,
		)
		return err
	case models.ScanCompleted, models.ScanFailed:
		_, err := s.db.ExecContext(ctx,
			`UPDATE scans SET status = $1, completed_at = $2, error = $3 WHERE id = $4`,
			status, now, errMsg, id,
		)
		return err
	default:
		_, err := s.db.ExecContext(ctx, `UPDATE scans SET status = $1 WHERE id = $2`, status, id)
		return err
	}
}

// SaveReport persists the drift report for a scan.
func (s *Store) SaveReport(ctx context.Context, report models.DriftReport) error {
	data, err := json.Marshal(report)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE scans SET report_json = $1 WHERE id = $2`, data, report.ScanID)
	return err
}

// GetScan retrieves a scan by ID.
func (s *Store) GetScan(ctx context.Context, id string) (*models.ScanRecord, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, status, provider, state_source, options_json, report_json, error, created_at, started_at, completed_at FROM scans WHERE id = $1`,
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
		`SELECT id, status, provider, state_source, options_json, report_json, error, created_at, started_at, completed_at FROM scans ORDER BY created_at DESC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

// ListScanSummaries returns recent scans without full report payloads.
func (s *Store) ListScanSummaries(ctx context.Context, limit int) ([]models.ScanSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, status, provider, state_source, options_json, error, created_at, started_at, completed_at,
       COALESCE((report_json->'summary'->>'total_drifts')::int, 0)
FROM scans ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []models.ScanSummary
	for rows.Next() {
		var summary models.ScanSummary
		var optsJSON []byte
		var errMsg sql.NullString
		var startedAt, completedAt sql.NullTime

		if err := rows.Scan(&summary.ID, &summary.Status, &summary.Provider, &summary.StateSource,
			&optsJSON, &errMsg, &summary.CreatedAt, &startedAt, &completedAt, &summary.TotalDrifts); err != nil {
			return nil, err
		}
		if startedAt.Valid {
			t := startedAt.Time
			summary.StartedAt = &t
		}
		if completedAt.Valid {
			t := completedAt.Time
			summary.CompletedAt = &t
		}
		if errMsg.Valid {
			summary.Error = errMsg.String
		}
		if len(optsJSON) > 0 {
			var opts models.ScanOptions
			if err := json.Unmarshal(optsJSON, &opts); err == nil {
				summary.ProfileName = opts.ProfileName
			}
		}
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}

func scanRow(row *sql.Row) (*models.ScanRecord, error) {
	var rec models.ScanRecord
	var optsJSON, reportJSON []byte
	var errMsg sql.NullString
	var startedAt, completedAt sql.NullTime

	err := row.Scan(&rec.ID, &rec.Status, &rec.Provider, &rec.StateSource, &optsJSON, &reportJSON, &errMsg, &rec.CreatedAt, &startedAt, &completedAt)
	if err != nil {
		return nil, err
	}
	if startedAt.Valid {
		t := startedAt.Time
		rec.StartedAt = &t
	}
	if completedAt.Valid {
		t := completedAt.Time
		rec.CompletedAt = &t
	}
	if errMsg.Valid {
		rec.Error = errMsg.String
	}
	if len(optsJSON) > 0 {
		_ = json.Unmarshal(optsJSON, &rec.Options)
	}
	if len(reportJSON) > 0 {
		var report models.DriftReport
		if err := json.Unmarshal(reportJSON, &report); err == nil {
			rec.Report = &report
		}
	}
	return &rec, nil
}

func scanRows(rows *sql.Rows) ([]models.ScanRecord, error) {
	var records []models.ScanRecord
	for rows.Next() {
		var rec models.ScanRecord
		var optsJSON, reportJSON []byte
		var errMsg sql.NullString
		var startedAt, completedAt sql.NullTime

		if err := rows.Scan(&rec.ID, &rec.Status, &rec.Provider, &rec.StateSource, &optsJSON, &reportJSON, &errMsg, &rec.CreatedAt, &startedAt, &completedAt); err != nil {
			return nil, err
		}
		if startedAt.Valid {
			t := startedAt.Time
			rec.StartedAt = &t
		}
		if completedAt.Valid {
			t := completedAt.Time
			rec.CompletedAt = &t
		}
		if errMsg.Valid {
			rec.Error = errMsg.String
		}
		if len(optsJSON) > 0 {
			_ = json.Unmarshal(optsJSON, &rec.Options)
		}
		if len(reportJSON) > 0 {
			var report models.DriftReport
			if err := json.Unmarshal(reportJSON, &report); err == nil {
				rec.Report = &report
			}
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

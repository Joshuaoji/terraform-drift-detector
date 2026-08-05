package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/terraform-drift-detector/driftdetect/internal/scan"
	"github.com/terraform-drift-detector/driftdetect/internal/store"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// Server exposes the drift detection REST API.
type Server struct {
	scanner *scan.Scanner
	store   store.Store
	router  chi.Router
}

// NewServer creates an API server.
func NewServer(scanner *scan.Scanner, st store.Store) *Server {
	s := &Server{scanner: scanner, store: st}
	s.router = chi.NewRouter()
	s.router.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer, middleware.Timeout(60*time.Second))
	s.routes()
	return s
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) routes() {
	s.router.Get("/health", s.handleHealth)
	s.router.Route("/api/v1", func(r chi.Router) {
		r.Post("/scans", s.handleCreateScan)
		r.Get("/scans", s.handleListScans)
		r.Get("/scans/{id}", s.handleGetScan)
		r.Get("/scans/{id}/report", s.handleGetReport)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type createScanRequest struct {
	Provider      models.Provider     `json:"provider"`
	StateSource   models.StateSource  `json:"state_source"`
	StatePath     string              `json:"state_path,omitempty"`
	Regions       []string            `json:"regions,omitempty"`
	ResourceTypes []string            `json:"resource_types,omitempty"`
	AccountID     string              `json:"account_id,omitempty"`
	ProjectID     string              `json:"project_id,omitempty"`
}

func (s *Server) handleCreateScan(w http.ResponseWriter, r *http.Request) {
	var req createScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	opts := models.ScanOptions{
		StateSource:   req.StateSource,
		StatePath:     req.StatePath,
		Provider:      req.Provider,
		Regions:       req.Regions,
		ResourceTypes: req.ResourceTypes,
		AccountID:     req.AccountID,
		ProjectID:     req.ProjectID,
	}
	if opts.ResolvedStateSource().Display() == "" {
		writeError(w, http.StatusBadRequest, "state_source or state_path is required")
		return
	}
	if opts.Provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}

	record, err := s.store.CreateScan(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	go s.runScan(record.ID, opts)

	writeJSON(w, http.StatusAccepted, record)
}

func (s *Server) runScan(id string, opts models.ScanOptions) {
	ctx := context.Background()
	_ = s.store.UpdateStatus(ctx, id, models.ScanRunning, "")
	report, err := s.scanner.Run(ctx, opts)
	if err != nil {
		_ = s.store.UpdateStatus(ctx, id, models.ScanFailed, err.Error())
		return
	}
	report.ScanID = id
	_ = s.store.SaveReport(ctx, report)
	_ = s.store.UpdateStatus(ctx, id, models.ScanCompleted, "")
}

func (s *Server) handleListScans(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	records, err := s.store.ListScans(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if records == nil {
		records = []models.ScanRecord{}
	}
	writeJSON(w, http.StatusOK, records)
}

func (s *Server) handleGetScan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	record, err := s.store.GetScan(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "scan not found")
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *Server) handleGetReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	record, err := s.store.GetScan(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "scan not found")
		return
	}
	if record.Report == nil {
		writeError(w, http.StatusNotFound, "report not available")
		return
	}
	writeJSON(w, http.StatusOK, record.Report)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

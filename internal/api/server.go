package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/terraform-drift-detector/driftdetect/internal/config"
	"github.com/terraform-drift-detector/driftdetect/internal/scan"
	"github.com/terraform-drift-detector/driftdetect/internal/scheduler"
	"github.com/terraform-drift-detector/driftdetect/internal/store"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// Server exposes the drift detection REST API and web dashboard.
type Server struct {
	service   *scan.Service
	store     store.Store
	scheduler *scheduler.Scheduler
	config    *config.Config
	router    chi.Router
}

// NewServer creates an API server.
func NewServer(service *scan.Service, st store.Store, sched *scheduler.Scheduler, cfg *config.Config) *Server {
	s := &Server{service: service, store: st, scheduler: sched, config: cfg}
	s.router = chi.NewRouter()
	s.router.Use(corsMiddleware, middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer, middleware.Timeout(120*time.Second))
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
		r.Get("/profiles", s.handleListProfiles)
		r.Post("/scans", s.handleCreateScan)
		r.Post("/scans/profile/{name}", s.handleCreateScanFromProfile)
		r.Get("/scans", s.handleListScans)
		r.Get("/scans/{id}", s.handleGetScan)
		r.Get("/scans/{id}/report", s.handleGetReport)
	})
	s.router.Handle("/*", webHandler())
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	if s.config == nil {
		writeJSON(w, http.StatusOK, []models.ScanProfileInfo{})
		return
	}
	profiles := make([]models.ScanProfileInfo, 0, len(s.config.Scans))
	for _, p := range s.config.Scans {
		profiles = append(profiles, models.ScanProfileInfo{
			Name:          p.Name,
			Provider:      p.Provider,
			Regions:       p.Regions,
			ResourceTypes: p.ResourceTypes,
			Schedule:      p.Schedule,
			StateSource:   p.ToScanOptions().ResolvedStateSource().Display(),
		})
	}
	writeJSON(w, http.StatusOK, profiles)
}

type createScanRequest struct {
	Provider      models.Provider    `json:"provider"`
	StateSource   models.StateSource `json:"state_source"`
	StatePath     string             `json:"state_path,omitempty"`
	Regions       []string           `json:"regions,omitempty"`
	ResourceTypes []string           `json:"resource_types,omitempty"`
	AccountID     string             `json:"account_id,omitempty"`
	ProjectID     string             `json:"project_id,omitempty"`
	ProfileName   string             `json:"profile_name,omitempty"`
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
		ProfileName:   req.ProfileName,
	}
	if err := validateScanOptions(opts); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	record, err := s.service.RunAsync(opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, record)
}

func (s *Server) handleCreateScanFromProfile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if s.config == nil {
		writeError(w, http.StatusNotFound, "no config loaded")
		return
	}
	profile, err := s.config.FindScan(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var record *models.ScanRecord
	if s.scheduler != nil {
		record, err = s.scheduler.TriggerNow(*profile)
	} else {
		opts := profile.ToScanOptions()
		opts.ProfileName = profile.Name
		record, err = s.service.RunAsync(opts)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, record)
}

func (s *Server) handleListScans(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if r.URL.Query().Get("summary") == "true" {
		summaries, err := s.store.ListScanSummaries(r.Context(), limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if summaries == nil {
			summaries = []models.ScanSummary{}
		}
		writeJSON(w, http.StatusOK, summaries)
		return
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

func validateScanOptions(opts models.ScanOptions) error {
	if opts.ResolvedStateSource().Display() == "" {
		return errString("state_source or state_path is required")
	}
	if opts.Provider == "" {
		return errString("provider is required")
	}
	return nil
}

type errString string

func (e errString) Error() string { return string(e) }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

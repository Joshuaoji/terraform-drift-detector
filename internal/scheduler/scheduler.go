package scheduler

import (
	"fmt"
	"log"
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/terraform-drift-detector/driftdetect/internal/config"
	"github.com/terraform-drift-detector/driftdetect/internal/scan"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// Scheduler runs scan profiles on cron schedules.
type Scheduler struct {
	cron    *cron.Cron
	service *scan.Service
	mu      sync.Mutex
	running map[string]bool
}

// New creates a scheduler bound to a scan service.
func New(service *scan.Service) *Scheduler {
	return &Scheduler{
		cron:    cron.New(),
		service: service,
		running: make(map[string]bool),
	}
}

// LoadProfiles registers cron jobs for profiles that define a schedule.
func (s *Scheduler) LoadProfiles(profiles []config.ScanProfile) error {
	for _, profile := range profiles {
		if profile.Schedule == "" {
			continue
		}
		p := profile
		expr := p.Schedule
		_, err := s.cron.AddFunc(expr, func() {
			s.triggerProfile(p)
		})
		if err != nil {
			return fmt.Errorf("profile %q schedule %q: %w", p.Name, expr, err)
		}
		log.Printf("scheduled scan profile %q with cron %q", p.Name, expr)
	}
	return nil
}

func (s *Scheduler) triggerProfile(profile config.ScanProfile) {
	s.mu.Lock()
	if s.running[profile.Name] {
		s.mu.Unlock()
		log.Printf("skipping scheduled scan %q: previous run still in progress", profile.Name)
		return
	}
	s.running[profile.Name] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.running, profile.Name)
		s.mu.Unlock()
	}()

	opts := profile.ToScanOptions()
	opts.ProfileName = profile.Name
	log.Printf("starting scheduled scan for profile %q", profile.Name)
	if _, err := s.service.RunAsync(opts); err != nil {
		log.Printf("scheduled scan %q failed to start: %v", profile.Name, err)
	}
}

// Start begins the cron scheduler.
func (s *Scheduler) Start() {
	s.cron.Start()
}

// Stop halts the cron scheduler.
func (s *Scheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
}

// TriggerNow runs a named profile immediately (used by API).
func (s *Scheduler) TriggerNow(profile config.ScanProfile) (*models.ScanRecord, error) {
	opts := profile.ToScanOptions()
	opts.ProfileName = profile.Name
	return s.service.RunAsync(opts)
}

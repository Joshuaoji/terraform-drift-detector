package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/terraform-drift-detector/driftdetect/internal/observability"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// Event names for webhook delivery.
const (
	EventScanCompleted = "scan.completed"
	EventScanFailed    = "scan.failed"
)

// Config defines a webhook endpoint.
type Config struct {
	Name   string   `yaml:"name"`
	URL    string   `yaml:"url"`
	Events []string `yaml:"events"`
	Secret string   `yaml:"secret,omitempty"`
}

// Payload is sent to webhook receivers.
type Payload struct {
	Event     string              `json:"event"`
	ScanID    string              `json:"scan_id"`
	Status    models.ScanStatus     `json:"status"`
	Provider  models.Provider       `json:"provider"`
	Profile   string              `json:"profile_name,omitempty"`
	Summary   *models.DriftSummary  `json:"summary,omitempty"`
	Error     string              `json:"error,omitempty"`
	Timestamp time.Time           `json:"timestamp"`
}

// Notifier delivers webhook events.
type Notifier struct {
	webhooks []Config
	client   *http.Client
	log      *slog.Logger
}

// New creates a webhook notifier.
func New(webhooks []Config, log *slog.Logger) *Notifier {
	if log == nil {
		log = slog.Default()
	}
	return &Notifier{
		webhooks: webhooks,
		client:   &http.Client{Timeout: 15 * time.Second},
		log:      log,
	}
}

// Enabled returns true when webhooks are configured.
func (n *Notifier) Enabled() bool {
	return len(n.webhooks) > 0
}

// NotifyScanCompleted sends scan.completed events.
func (n *Notifier) NotifyScanCompleted(record *models.ScanRecord, report models.DriftReport) {
	if !n.Enabled() || record == nil {
		return
	}
	payload := Payload{
		Event:     EventScanCompleted,
		ScanID:    record.ID,
		Status:    record.Status,
		Provider:  record.Provider,
		Profile:   record.Options.ProfileName,
		Summary:   &report.Summary,
		Timestamp: time.Now().UTC(),
	}
	n.dispatch(EventScanCompleted, payload)
}

// NotifyScanFailed sends scan.failed events.
func (n *Notifier) NotifyScanFailed(record *models.ScanRecord, errMsg string) {
	if !n.Enabled() || record == nil {
		return
	}
	payload := Payload{
		Event:     EventScanFailed,
		ScanID:    record.ID,
		Status:    models.ScanFailed,
		Provider:  record.Provider,
		Profile:   record.Options.ProfileName,
		Error:     errMsg,
		Timestamp: time.Now().UTC(),
	}
	n.dispatch(EventScanFailed, payload)
}

func (n *Notifier) dispatch(event string, payload Payload) {
	for _, wh := range n.webhooks {
		if !matchesEvent(wh.Events, event) {
			continue
		}
		go n.send(wh, payload)
	}
}

func (n *Notifier) send(wh Config, payload Payload) {
	body, err := json.Marshal(payload)
	if err != nil {
		n.log.Error("webhook marshal failed", "name", wh.Name, "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		n.log.Error("webhook request failed", "name", wh.Name, "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "driftdetect/0.4")
	if wh.Secret != "" {
		mac := hmac.New(sha256.New, []byte(wh.Secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Driftdetect-Signature", "sha256="+sig)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		observability.RecordWebhookDelivery(false)
		n.log.Error("webhook delivery failed", "name", wh.Name, "url", wh.URL, "error", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		observability.RecordWebhookDelivery(false)
		n.log.Warn("webhook non-success status", "name", wh.Name, "status", resp.StatusCode)
		return
	}
	observability.RecordWebhookDelivery(true)
	n.log.Info("webhook delivered", "name", wh.Name, "event", payload.Event, "scan_id", payload.ScanID)
}

func matchesEvent(events []string, event string) bool {
	if len(events) == 0 {
		return true
	}
	for _, e := range events {
		if e == event || e == "*" {
			return true
		}
	}
	return false
}

// Validate checks webhook configuration.
func Validate(cfgs []Config) error {
	for _, wh := range cfgs {
		if wh.URL == "" {
			return fmt.Errorf("webhook %q missing url", wh.Name)
		}
	}
	return nil
}

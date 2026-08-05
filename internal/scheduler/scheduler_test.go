package scheduler_test

import (
	"testing"

	"github.com/robfig/cron/v3"
	"github.com/terraform-drift-detector/driftdetect/internal/config"
)

func TestCronExpression_Valid(t *testing.T) {
	profiles := []config.ScanProfile{
		{Name: "prod", Schedule: "0 */6 * * *"},
		{Name: "daily", Schedule: "0 0 * * *"},
	}
	c := cron.New()
	for _, p := range profiles {
		if _, err := c.AddFunc(p.Schedule, func() {}); err != nil {
			t.Fatalf("invalid cron for %q: %v", p.Name, err)
		}
	}
}

func TestCronExpression_Invalid(t *testing.T) {
	c := cron.New()
	if _, err := c.AddFunc("not a cron", func() {}); err == nil {
		t.Fatal("expected error for invalid cron expression")
	}
}

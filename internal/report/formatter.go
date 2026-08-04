package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

// Format identifies output format.
type Format string

const (
	FormatConsole Format = "console"
	FormatJSON    Format = "json"
)

// Formatter renders drift reports.
type Formatter interface {
	Format(report models.DriftReport, w io.Writer) error
}

// ConsoleFormatter prints human-readable drift output.
type ConsoleFormatter struct{}

// Format writes a console report.
func (f *ConsoleFormatter) Format(report models.DriftReport, w io.Writer) error {
	fmt.Fprintf(w, "Drift Scan Report\n")
	fmt.Fprintf(w, "=================\n")
	fmt.Fprintf(w, "Scan ID:       %s\n", report.ScanID)
	fmt.Fprintf(w, "Provider:      %s\n", report.Provider)
	fmt.Fprintf(w, "State Source:  %s\n", report.StateSource)
	fmt.Fprintf(w, "Started:       %s\n", report.StartedAt.Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(w, "Completed:     %s\n", report.CompletedAt.Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintf(w, "\nSummary\n")
	fmt.Fprintf(w, "-------\n")
	fmt.Fprintf(w, "Total Drifts:      %d\n", report.Summary.TotalDrifts)
	fmt.Fprintf(w, "Missing in Cloud:  %d\n", report.Summary.MissingInCloud)
	fmt.Fprintf(w, "Missing in State:  %d\n", report.Summary.MissingInState)
	fmt.Fprintf(w, "Attribute Drifts:  %d\n", report.Summary.AttributeDrifts)
	fmt.Fprintf(w, "Tag Drifts:        %d\n", report.Summary.TagDrifts)
	fmt.Fprintf(w, "\n")

	if len(report.Drifts) == 0 {
		fmt.Fprintf(w, "No drift detected.\n")
		return nil
	}

	fmt.Fprintf(w, "Drift Details\n")
	fmt.Fprintf(w, "-------------\n")
	for i, d := range report.Drifts {
		fmt.Fprintf(w, "\n[%d] %s - %s (%s)\n", i+1, d.Type, d.ResourceType, d.ResourceID)
		if d.TerraformRef != "" {
			fmt.Fprintf(w, "    Terraform: %s\n", d.TerraformRef)
		}
		if d.Region != "" {
			fmt.Fprintf(w, "    Region:    %s\n", d.Region)
		}
		if d.Message != "" {
			fmt.Fprintf(w, "    Message:   %s\n", d.Message)
		}
		for _, ac := range d.AttributeChanges {
			fmt.Fprintf(w, "    Attribute: %s\n", ac.Attribute)
			fmt.Fprintf(w, "      Expected: %v\n", ac.Expected)
			fmt.Fprintf(w, "      Actual:   %v\n", ac.Actual)
		}
		for _, tc := range d.TagChanges {
			switch tc.Change {
			case "added":
				fmt.Fprintf(w, "    Tag added:   %s = %s\n", tc.Key, tc.Actual)
			case "removed":
				fmt.Fprintf(w, "    Tag removed: %s (was %s)\n", tc.Key, tc.Expected)
			case "modified":
				fmt.Fprintf(w, "    Tag changed: %s: %s -> %s\n", tc.Key, tc.Expected, tc.Actual)
			}
		}
	}
	return nil
}

// JSONFormatter writes machine-readable JSON output.
type JSONFormatter struct {
	Pretty bool
}

// Format writes a JSON report.
func (f *JSONFormatter) Format(report models.DriftReport, w io.Writer) error {
	var data []byte
	var err error
	if f.Pretty {
		data, err = json.MarshalIndent(report, "", "  ")
	} else {
		data, err = json.Marshal(report)
	}
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	_, err = w.Write(data)
	if !f.Pretty {
		_, _ = w.Write([]byte("\n"))
	} else {
		_, _ = w.Write([]byte("\n"))
	}
	return err
}

// NewFormatter returns a formatter for the given format string.
func NewFormatter(format string) (Formatter, error) {
	switch strings.ToLower(format) {
	case "console", "table", "":
		return &ConsoleFormatter{}, nil
	case "json":
		return &JSONFormatter{Pretty: true}, nil
	default:
		return nil, fmt.Errorf("unknown output format: %s", format)
	}
}

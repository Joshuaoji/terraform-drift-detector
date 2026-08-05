package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/terraform-drift-detector/driftdetect/internal/api"
	"github.com/terraform-drift-detector/driftdetect/internal/config"
	"github.com/terraform-drift-detector/driftdetect/internal/report"
	"github.com/terraform-drift-detector/driftdetect/internal/scan"
	"github.com/terraform-drift-detector/driftdetect/internal/scheduler"
	"github.com/terraform-drift-detector/driftdetect/internal/store/sqlite"
	"github.com/terraform-drift-detector/driftdetect/pkg/models"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "scan":
		os.Exit(runScan(os.Args[2:]))
	case "serve":
		os.Exit(runServe(os.Args[2:]))
	case "version":
		fmt.Println("driftdetect v0.3.0")
		os.Exit(0)
	case "help", "-h", "--help":
		printUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runScan(args []string) int {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	statePath := fs.String("state", "", "Path or URI to terraform state (local, s3://, gcs://, azure://)")
	provider := fs.String("provider", "aws", "Cloud provider: aws, azure, gcp")
	regions := fs.String("regions", "", "Comma-separated regions to scan")
	resourceTypes := fs.String("resource-types", "", "Comma-separated resource types to check")
	configPath := fs.String("config", "", "Path to YAML config file")
	scanName := fs.String("scan-name", "", "Named scan profile from config file")
	output := fs.String("output", "console", "Output format: console, json")
	accountID := fs.String("account-id", "", "Cloud account/subscription ID (optional)")
	projectID := fs.String("project-id", "", "GCP project ID (optional)")
	timeout := fs.Duration("timeout", 10*time.Minute, "Scan timeout")

	_ = fs.Parse(args)

	opts, err := resolveScanOptions(*statePath, *provider, *regions, *resourceTypes, *configPath, *scanName, *accountID, *projectID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	scanner := scan.NewScanner()
	driftReport, err := scanner.Run(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Scan failed: %v\n", err)
		return 1
	}

	formatter, err := report.NewFormatter(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if err := formatter.Format(driftReport, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Output error: %v\n", err)
		return 1
	}

	if driftReport.Summary.TotalDrifts > 0 {
		return 2
	}
	return 0
}

func runServe(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.String("port", "8080", "HTTP listen port")
	dbPath := fs.String("db", "driftdetect.db", "SQLite database path")
	configPath := fs.String("config", "", "Path to YAML config with scan profiles and schedules")
	_ = fs.Parse(args)

	st, err := sqlite.Open(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
		return 1
	}
	defer st.Close()

	var cfg *config.Config
	if *configPath != "" {
		cfg, err = config.Load(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			return 1
		}
	}

	scanner := scan.NewScanner()
	svc := scan.NewService(scanner, st)

	var sched *scheduler.Scheduler
	if cfg != nil {
		sched = scheduler.New(svc)
		if err := sched.LoadProfiles(cfg.Scans); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load schedules: %v\n", err)
			return 1
		}
		sched.Start()
		defer sched.Stop()
	}

	server := api.NewServer(svc, st, sched, cfg)
	addr := ":" + *port
	fmt.Printf("driftdetect server listening on %s\n", addr)
	fmt.Printf("Dashboard: http://localhost%s/\n", addr)
	if cfg != nil {
		fmt.Printf("Loaded %d scan profile(s) from %s\n", len(cfg.Scans), *configPath)
	}
	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		return 1
	}
	return 0
}

func resolveScanOptions(statePath, provider, regions, resourceTypes, configPath, scanName, accountID, projectID string) (models.ScanOptions, error) {
	if configPath != "" {
		cfg, err := config.Load(configPath)
		if err != nil {
			return models.ScanOptions{}, err
		}
		if scanName == "" {
			if len(cfg.Scans) == 0 {
				return models.ScanOptions{}, fmt.Errorf("no scan profiles in config")
			}
			scanName = cfg.Scans[0].Name
		}
		profile, err := cfg.FindScan(scanName)
		if err != nil {
			return models.ScanOptions{}, err
		}
		opts := profile.ToScanOptions()
		if statePath != "" {
			opts.StatePath = statePath
			opts.StateSource = models.ParseStatePath(statePath)
		}
		if accountID != "" {
			opts.AccountID = accountID
		}
		if projectID != "" {
			opts.ProjectID = projectID
		}
		return opts, nil
	}

	if statePath == "" {
		return models.ScanOptions{}, fmt.Errorf("--state is required (or use --config)")
	}

	return models.ScanOptions{
		StateSource:   models.ParseStatePath(statePath),
		StatePath:     statePath,
		Provider:      models.Provider(provider),
		Regions:       splitCSV(regions),
		ResourceTypes: splitCSV(resourceTypes),
		AccountID:     accountID,
		ProjectID:     projectID,
	}, nil
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func printUsage() {
	fmt.Println(`driftdetect - Terraform drift detection CLI

Usage:
  driftdetect scan [flags]
  driftdetect serve [flags]
  driftdetect version
  driftdetect help

Scan Flags:
  --state           Path or URI to terraform state (local, s3://, gcs://, azure://)
  --provider        Cloud provider (aws, azure, gcp) [default: aws]
  --regions         Comma-separated regions
  --resource-types  Comma-separated Terraform resource types
  --config          Path to YAML configuration file
  --scan-name       Named scan profile from config
  --output          Output format: console, json [default: console]
  --account-id      Cloud account/subscription ID
  --project-id      GCP project ID
  --timeout         Scan timeout [default: 10m]

Serve Flags:
  --port            HTTP listen port [default: 8080]
  --db              SQLite database path [default: driftdetect.db]
  --config          YAML config with scan profiles and cron schedules

API Endpoints:
  GET  /health
  GET  /api/v1/profiles
  POST /api/v1/scans
  POST /api/v1/scans/profile/{name}
  GET  /api/v1/scans?summary=true
  GET  /api/v1/scans/{id}
  GET  /api/v1/scans/{id}/report

Dashboard:
  Open http://localhost:8080/ when the server is running

Examples:
  driftdetect scan --state ./terraform.tfstate --provider aws --output json
  driftdetect scan --state s3://my-bucket/terraform.tfstate --provider aws
  driftdetect serve --port 8080 --config configs/example.yaml

Exit Codes (scan):
  0 - No drift detected
  1 - Error
  2 - Drift detected`)
}

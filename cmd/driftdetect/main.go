package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/terraform-drift-detector/driftdetect/internal/config"
	"github.com/terraform-drift-detector/driftdetect/internal/report"
	"github.com/terraform-drift-detector/driftdetect/internal/scan"
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
	case "version":
		fmt.Println("driftdetect v0.1.0")
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
	statePath := fs.String("state", "", "Path to terraform state file (required)")
	provider := fs.String("provider", "aws", "Cloud provider: aws, azure, gcp")
	regions := fs.String("regions", "", "Comma-separated regions to scan")
	resourceTypes := fs.String("resource-types", "", "Comma-separated resource types to check")
	configPath := fs.String("config", "", "Path to YAML config file")
	scanName := fs.String("scan-name", "", "Named scan profile from config file")
	output := fs.String("output", "console", "Output format: console, json")
	accountID := fs.String("account-id", "", "Cloud account ID (optional)")
	timeout := fs.Duration("timeout", 10*time.Minute, "Scan timeout")

	_ = fs.Parse(args)

	opts, err := resolveScanOptions(*statePath, *provider, *regions, *resourceTypes, *configPath, *scanName, *accountID)
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

func resolveScanOptions(statePath, provider, regions, resourceTypes, configPath, scanName, accountID string) (models.ScanOptions, error) {
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
		}
		if accountID != "" {
			opts.AccountID = accountID
		}
		return opts, nil
	}

	if statePath == "" {
		return models.ScanOptions{}, fmt.Errorf("--state is required (or use --config)")
	}

	return models.ScanOptions{
		StatePath:     statePath,
		Provider:      models.Provider(provider),
		Regions:       splitCSV(regions),
		ResourceTypes: splitCSV(resourceTypes),
		AccountID:     accountID,
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
  driftdetect version
  driftdetect help

Scan Flags:
  --state           Path to terraform state file
  --provider        Cloud provider (aws, azure, gcp) [default: aws]
  --regions         Comma-separated regions
  --resource-types  Comma-separated Terraform resource types
  --config          Path to YAML configuration file
  --scan-name       Named scan profile from config
  --output          Output format: console, json [default: console]
  --account-id      Cloud account ID
  --timeout         Scan timeout [default: 10m]

Examples:
  driftdetect scan --state ./terraform.tfstate --provider aws --output json
  driftdetect scan --config configs/example.yaml --scan-name prod-aws
  driftdetect scan --state ./state.json --resource-types aws_s3_bucket,aws_instance

Exit Codes:
  0 - No drift detected
  1 - Error
  2 - Drift detected`)
}

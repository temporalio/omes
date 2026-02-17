package promconfig

import (
	"fmt"
	"strings"
)

// Config holds the configuration for generating a Prometheus scrape config.
type Config struct {
	ScrapeInterval string   // e.g., "1s"
	ClientHost     string   // Client metrics endpoint (e.g., "localhost:9091")
	WorkerHosts    []string // Worker SDK metrics endpoints
	ProcessHosts   []string // Worker process metrics endpoints (from --remote-worker)
}

// Generate creates a Prometheus configuration YAML from the given config.
func Generate(cfg Config) ([]byte, error) {
	if cfg.ScrapeInterval == "" {
		cfg.ScrapeInterval = "1s"
	}
	cfg.WorkerHosts = filterHosts(cfg.WorkerHosts)
	cfg.ProcessHosts = filterHosts(cfg.ProcessHosts)

	var sb strings.Builder

	// Global config
	sb.WriteString("global:\n")
	sb.WriteString(fmt.Sprintf("  scrape_interval: %s\n", cfg.ScrapeInterval))
	sb.WriteString(fmt.Sprintf("  evaluation_interval: %s\n", cfg.ScrapeInterval))
	sb.WriteString("\n")

	// Scrape configs
	sb.WriteString("scrape_configs:\n")

	// Client metrics
	if cfg.ClientHost != "" {
		sb.WriteString("  - job_name: 'omes-client'\n")
		sb.WriteString("    static_configs:\n")
		sb.WriteString(fmt.Sprintf("      - targets: ['%s']\n", cfg.ClientHost))
	}

	// Worker SDK metrics
	if len(cfg.WorkerHosts) > 0 {
		sb.WriteString("\n")
		sb.WriteString("  - job_name: 'omes-worker'\n")
		sb.WriteString("    static_configs:\n")
		sb.WriteString("      - targets: [")
		for i, host := range cfg.WorkerHosts {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("'%s'", host))
		}
		sb.WriteString("]\n")
	}

	// Worker process metrics
	if len(cfg.ProcessHosts) > 0 {
		sb.WriteString("\n")
		sb.WriteString("  - job_name: 'omes-worker-process'\n")
		sb.WriteString("    static_configs:\n")
		sb.WriteString("      - targets: [")
		for i, host := range cfg.ProcessHosts {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("'%s'", host))
		}
		sb.WriteString("]\n")
	}

	return []byte(sb.String()), nil
}

func filterHosts(hosts []string) []string {
	filtered := make([]string, 0, len(hosts))
	for _, host := range hosts {
		host = strings.TrimSpace(host)
		if host != "" {
			filtered = append(filtered, host)
		}
	}
	return filtered
}

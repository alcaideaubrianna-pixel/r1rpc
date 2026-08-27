package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecutionTimingValidation(t *testing.T) {
	valid := Config{
		RequestTimeout:           25 * time.Second,
		QueueLeaseDuration:       10 * time.Second,
		QueueLeaseReaperInterval: time.Second,
		ExecutionGrace:           30 * time.Second,
	}
	if err := valid.ValidateExecutionTiming(); err != nil {
		t.Fatalf("valid timing: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{name: "lease exceeds request", mutate: func(c *Config) { c.QueueLeaseDuration = 30 * time.Second }, want: "request timeout"},
		{name: "reaper not below lease", mutate: func(c *Config) { c.QueueLeaseReaperInterval = c.QueueLeaseDuration }, want: "reaper"},
		{name: "grace below reaper", mutate: func(c *Config) { c.ExecutionGrace = 500 * time.Millisecond }, want: "execution grace"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := valid
			test.mutate(&cfg)
			if err := cfg.ValidateExecutionTiming(); err == nil || !strings.Contains(strings.ToLower(err.Error()), test.want) {
				t.Fatalf("error=%v want substring %q", err, test.want)
			}
		})
	}
}

func TestParseExecutionGraceFromYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("limits:\n  execution_grace_seconds: 45\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	values, _, err := parseConfigFile(path)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if got := values["EXECUTION_GRACE_SECONDS"]; got != "45" {
		t.Fatalf("execution grace=%q", got)
	}
}

func TestExecutionGraceYAMLMapping(t *testing.T) {
	found := false
	for _, mapping := range yamlToEnvKey {
		if mapping.Section == "limits" && mapping.Key == "execution_grace_seconds" && mapping.Env == "EXECUTION_GRACE_SECONDS" {
			found = true
		}
	}
	if !found {
		t.Fatal("execution_grace_seconds mapping missing")
	}
}

func TestClientQueueIdleTTLYAMLMapping(t *testing.T) {
	found := false
	for _, mapping := range yamlToEnvKey {
		if mapping.Section == "limits" && mapping.Key == "client_queue_idle_ttl_seconds" && mapping.Env == "CLIENT_QUEUE_IDLE_TTL_SECONDS" {
			found = true
		}
	}
	if !found {
		t.Fatal("client_queue_idle_ttl_seconds mapping missing")
	}
}

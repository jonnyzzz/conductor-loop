package config

import "strings"

// APIConfig defines the REST API configuration settings.
type APIConfig struct {
	Host        string    `yaml:"host"`
	Port        int       `yaml:"port"`
	CORSOrigins []string  `yaml:"cors_origins"`
	AuthEnabled bool      `yaml:"auth_enabled"`
	SSE         SSEConfig `yaml:"sse"`
}

// SSEConfig defines server-sent events configuration.
type SSEConfig struct {
	PollIntervalMs      int `yaml:"poll_interval_ms"`
	DiscoveryIntervalMs int `yaml:"discovery_interval_ms"`
	HeartbeatIntervalS  int `yaml:"heartbeat_interval_s"`
	MaxClientsPerRun    int `yaml:"max_clients_per_run"`
}

func applyAPIDefaults(cfg *Config) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.API.Host) == "" {
		cfg.API.Host = "0.0.0.0"
	}
	if cfg.API.Port == 0 {
		cfg.API.Port = 8080
	}
	if cfg.API.SSE.PollIntervalMs == 0 {
		cfg.API.SSE.PollIntervalMs = 100
	}
	if cfg.API.SSE.DiscoveryIntervalMs == 0 {
		cfg.API.SSE.DiscoveryIntervalMs = 1000
	}
	if cfg.API.SSE.HeartbeatIntervalS == 0 {
		cfg.API.SSE.HeartbeatIntervalS = 30
	}
	if cfg.API.SSE.MaxClientsPerRun == 0 {
		cfg.API.SSE.MaxClientsPerRun = 10
	}
}

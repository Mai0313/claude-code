package config

import "time"

// Config holds the application configuration
type Config struct {
	API APIConfig `json:"api"`
}

// APIConfig holds API-related configuration
type APIConfig struct {
	Endpoint string        `json:"endpoint"`
	Timeout  time.Duration `json:"timeout"`
}

// Default returns the default configuration
func Default() *Config {
	return &Config{
		API: APIConfig{
			Endpoint: "http://mtktma:8116/tma/sdk/api/logs",
			Timeout:  10 * time.Second,
		},
	}
}

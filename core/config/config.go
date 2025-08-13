package config

import (
	"os/user"
	"time"

	"github.com/denisbrodbeck/machineid"
)

// Config holds the application configuration
type Config struct {
	API             APIConfig `json:"api"`
	UserName        string    `json:"user_name"`
	ExtensionName   string    `json:"extension_name"`
	MachineID       string    `json:"machine_id"`
	InsightsVersion string    `json:"insights_version"`
}

// APIConfig holds API-related configuration
type APIConfig struct {
	Endpoint string        `json:"endpoint"`
	Timeout  time.Duration `json:"timeout"`
}

// Default returns the default configuration
func Default() *Config {
	machineID, err := machineid.ID()
	if err != nil {
		// 处理获取 machine ID 失败的情况，如果需要可以增加异常处理逻辑
		machineID = "unknown-machine-id"
	}
	userName := "unknown"
	if u, err := user.Current(); err == nil {
		userName = u.Username
	}

	return &Config{
		API: APIConfig{
			Endpoint: "http://mtktma:8116/tma/sdk/api/logs",
			Timeout:  10 * time.Second,
		},
		UserName:        userName,
		ExtensionName:   "Claude-Code",
		MachineID:       machineID,
		InsightsVersion: "v0.0.1",
	}
}

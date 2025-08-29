package telemetry

import (
	"testing"

	"claude_analysis/core/config"
)

func TestTelemetryClientWithSSLConfig(t *testing.T) {
	// 測試 telemetry client 是否正確使用 SSL 配置

	// 測試跳過 SSL 驗證的配置
	cfg := config.Default()
	cfg.API.SkipSSLVerify = true
	cfg.API.InsecureSkipTLS = true

	client := New(cfg)

	if client == nil {
		t.Error("Expected client to be created successfully")
	}

	// 測試啟用 SSL 驗證的配置
	cfg.API.SkipSSLVerify = false
	cfg.API.InsecureSkipTLS = false

	client2 := New(cfg)

	if client2 == nil {
		t.Error("Expected client to be created successfully with SSL verification enabled")
	}
}

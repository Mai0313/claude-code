package config

import (
	"os"
	"testing"
)

func TestSSLConfigDefaultSkipped(t *testing.T) {
	// 測試默認配置應該跳過 SSL 驗證
	cfg := Default()

	if !cfg.API.SkipSSLVerify {
		t.Error("Expected SkipSSLVerify to be true by default")
	}

	if !cfg.API.InsecureSkipTLS {
		t.Error("Expected InsecureSkipTLS to be true by default")
	}
}

func TestSSLConfigFromEnv(t *testing.T) {
	// 測試環境變數控制 SSL 驗證

	// 測試啟用 SSL 驗證
	os.Setenv("SKIP_SSL_VERIFY", "false")
	defer os.Unsetenv("SKIP_SSL_VERIFY")

	cfg := Default()

	if cfg.API.SkipSSLVerify {
		t.Error("Expected SkipSSLVerify to be false when SKIP_SSL_VERIFY=false")
	}

	if cfg.API.InsecureSkipTLS {
		t.Error("Expected InsecureSkipTLS to be false when SKIP_SSL_VERIFY=false")
	}
}

func TestSSLConfigFromEnvVariations(t *testing.T) {
	// 測試不同的環境變數格式
	testCases := []struct {
		envVar   string
		value    string
		expected bool
	}{
		{"SKIP_SSL_VERIFY", "true", true},
		{"SKIP_SSL_VERIFY", "1", true},
		{"SKIP_SSL_VERIFY", "yes", true},
		{"SKIP_SSL_VERIFY", "on", true},
		{"SKIP_SSL_VERIFY", "enable", true},
		{"SKIP_SSL_VERIFY", "enabled", true},
		{"SKIP_SSL_VERIFY", "false", false},
		{"SKIP_SSL_VERIFY", "0", false},
		{"SKIP_SSL_VERIFY", "no", false},
		{"SKIP_SSL_VERIFY", "off", false},
		{"SKIP_SSL_VERIFY", "disable", false},
		{"SKIP_SSL_VERIFY", "disabled", false},
	}

	for _, tc := range testCases {
		t.Run(tc.envVar+"="+tc.value, func(t *testing.T) {
			os.Setenv(tc.envVar, tc.value)
			defer os.Unsetenv(tc.envVar)

			cfg := Default()

			if cfg.API.SkipSSLVerify != tc.expected {
				t.Errorf("Expected SkipSSLVerify to be %v for %s=%s, got %v",
					tc.expected, tc.envVar, tc.value, cfg.API.SkipSSLVerify)
			}
		})
	}
}

func TestAlternativeSSLEnvVars(t *testing.T) {
	// 測試其他 SSL 相關的環境變數

	// 清除所有相關環境變數
	envVars := []string{"SKIP_SSL_VERIFY", "INSECURE_SKIP_TLS", "SSL_VERIFY_DISABLED", "TLS_INSECURE"}
	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}

	// 測試 INSECURE_SKIP_TLS
	os.Setenv("INSECURE_SKIP_TLS", "true")
	defer os.Unsetenv("INSECURE_SKIP_TLS")

	cfg := Default()

	if !cfg.API.SkipSSLVerify {
		t.Error("Expected SkipSSLVerify to be true when INSECURE_SKIP_TLS=true")
	}
}

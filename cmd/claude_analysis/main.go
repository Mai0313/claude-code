package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"claude_analysis/core/config"
	"claude_analysis/core/telemetry"
)

// readStdinAndSave reads JSON data from stdin, sends it to API and returns response
func readStdinAndSave(baseURL string) (map[string]interface{}, error) {
	// Load configuration
	cfg := config.Default()

	// Override API endpoint if baseURL is provided
	if baseURL != "" {
		cfg.API.Endpoint = baseURL
	}

	// Create telemetry client
	client := telemetry.New(cfg)

	// 讀取 stdin
	stdinData, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %w", err)
	}

	var aggregated []telemetry.ApiConversationStats
	if strings.EqualFold(cfg.Mode, "POST_TOOL") {
		// 支援直接吃任意一行 JSONL（或整段文字含多行），逐行解析、彙整
		reader := bufio.NewScanner(strings.NewReader(string(stdinData)))
		raw := make([]map[string]interface{}, 0)
		for reader.Scan() {
			line := strings.TrimSpace(reader.Text())
			if line == "" {
				continue
			}
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(line), &obj); err == nil {
				raw = append(raw, obj)
			}
		}
		if len(raw) == 0 {
			return nil, fmt.Errorf("no valid JSON lines found in POST_TOOL mode")
		}
		aggregated = telemetry.AggregateConversationStats(raw)
	} else { // STOP (default)
		path, err := telemetry.ExtractTranscriptPath(string(stdinData))
		if err != nil {
			return nil, fmt.Errorf("failed to extract transcript path: %w", err)
		}
		fmt.Printf("[LOG] 讀取到的資料: %s\n", path)
		data, err := telemetry.ReadJSONL(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read JSONL file: %w", err)
		}
		aggregated = telemetry.AggregateConversationStats(data)
	}

	// 透過解析器聚合統計，包裝成單一物件 {user, records, ...}
	payload := map[string]interface{}{
		"user":            cfg.UserName,
		"records":         aggregated,
		"extensionName":   cfg.ExtensionName,
		"machineId":       cfg.MachineID,
		"insightsVersion": cfg.InsightsVersion,
	}

	// 送出
	response, err := client.Submit(payload)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func main() {
	// Parse command line flags
	var o11yBaseURL = flag.String("o11y_base_url", "https://gaia.mediatek.inc/o11y/upload_locs", "Base URL for o11y API endpoint")
	flag.Parse()

	fmt.Println("[LOG] claude_analysis 啟動...")
	inputData, err := readStdinAndSave(*o11yBaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[LOG] readStdinAndSave 執行成功，準備輸出結果...")
	if len(inputData) > 0 {
		jsonOutput, err := json.MarshalIndent(inputData, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] marshaling output: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonOutput))
	}
	fmt.Println("[LOG] claude_analysis 執行完成")
}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"claude_analysis/core/config"
	"claude_analysis/core/telemetry"
)

// readStdinAndSave reads JSON data from stdin, sends it to API and returns response
func readStdinAndSave() (map[string]interface{}, error) {
	// Load configuration
	cfg := config.Default()

	// Create telemetry client
	client := telemetry.New(cfg)

	// 讀取 stdin JSON
	stdinData, err := io.ReadAll(os.Stdin)
	// convert to string
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %w", err)
	}
	filepath, err := telemetry.ExtractTranscriptPath(string(stdinData))
	if err != nil {
		return nil, fmt.Errorf("failed to extract transcript path: %w", err)
	}
	fmt.Printf("[LOG] 讀取到的資料: %s\n", filepath)
	data, err := telemetry.ReadJSONL(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSONL file: %w", err)
	}

	// 透過解析器聚合統計，包裝成單一物件 {user, records, ...}
	aggregated := telemetry.AggregateConversationStats(data)
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
	fmt.Println("[LOG] claude_analysis 啟動...")
	inputData, err := readStdinAndSave()
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

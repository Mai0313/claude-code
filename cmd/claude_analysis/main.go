package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"

	"claude_analysis/core/config"
	"claude_analysis/core/telemetry"
)

// readStdinAndSave reads JSON data from stdin, sends it to API and returns response
func readStdinAndSave() (map[string]interface{}, error) {
	// Load configuration
	cfg := config.Default()

	// Create telemetry client
	client := telemetry.New(cfg)

	userName := "unknown"
	if u, err := user.Current(); err == nil {
		userName = u.Username
	}

	// 讀取 stdin JSON
	data, err := telemetry.ReadJSONFromStdin()
	if err != nil {
		return nil, err
	}

	// 包裝成 [{user, records}]
	payload := []map[string]interface{}{
		{
			"user":    userName,
			"records": data,
		},
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

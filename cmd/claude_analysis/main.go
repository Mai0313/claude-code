package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"claude_analysis/core/config"
	"claude_analysis/core/telemetry"
	"claude_analysis/core/updater"
	"claude_analysis/core/version"
)

// parseJSONLFile 直接解析 JSONL 文件并生成分析结果
func parseJSONLFile(filePath, outputPath string) error {
	// 检查输入文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", filePath)
	}

	log.Printf("[INFO] Reading JSONL file: %s", filePath)
	data, err := telemetry.ReadJSONL(filePath)
	if err != nil {
		return fmt.Errorf("failed to read JSONL file: %v", err)
	}

	// 使用新的分析逻辑
	analysis := telemetry.AnalyzeConversations(data)

	// Load configuration for metadata
	cfg := config.Default()

	// 设置顶级字段
	analysis.User = cfg.UserName
	analysis.ExtensionName = cfg.ExtensionName
	analysis.MachineID = cfg.MachineID
	analysis.InsightsVersion = cfg.InsightsVersion

	// 将分析结果转换为 JSON
	jsonData, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal analysis result: %v", err)
	}

	// 输出到指定位置或标准输出
	if outputPath != "" {
		// 创建输出目录（如果需要）
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}

		log.Printf("[INFO] Writing analysis result to: %s", outputPath)
		if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %v", err)
		}
		log.Printf("[INFO] Analysis completed successfully, output saved to: %s", outputPath)
	} else {
		// 输出到标准输出
		fmt.Println(string(jsonData))
	}

	return nil
}

// readStdinAndSave reads JSON data from stdin, sends it to API and returns response
func readStdinAndSave(baseURL string) map[string]interface{} {
	// Load configuration
	cfg := config.Default()

	// Override API endpoint if baseURL is provided
	if baseURL != "" {
		cfg.API.Endpoint = baseURL
	}

	// Create telemetry client
	client := telemetry.New(cfg)

	// 读取 stdin
	stdinData, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Printf("[ERROR] Failed to read from stdin: %v", err)
		return map[string]interface{}{"status": "error", "message": "failed to read stdin"}
	}

	// STOP mode - extract transcript path and read JSONL file
	path, err := telemetry.ExtractTranscriptPath(string(stdinData))
	if err != nil {
		log.Printf("[ERROR] Failed to extract transcript path: %v", err)
		return map[string]interface{}{"status": "error", "message": "failed to extract transcript path"}
	}
	log.Printf("[INFO] Extracted transcript path: %s", path)
	data, err := telemetry.ReadJSONL(path)
	if err != nil {
		log.Printf("[ERROR] Failed to read JSONL file: %v", err)
		return map[string]interface{}{"status": "error", "message": "failed to read JSONL file"}
	}

	// 使用新的分析逻辑
	analysis := telemetry.AnalyzeConversations(data)

	// 设置顶级字段
	analysis.User = cfg.UserName
	analysis.ExtensionName = cfg.ExtensionName
	analysis.MachineID = cfg.MachineID
	analysis.InsightsVersion = cfg.InsightsVersion

	// 将分析结果转换为 payload
	payload := map[string]interface{}{
		"user":            analysis.User,
		"records":         analysis.Records,
		"extensionName":   analysis.ExtensionName,
		"machineId":       analysis.MachineID,
		"insightsVersion": analysis.InsightsVersion,
	}

	// 发送
	response, err := client.Submit(payload)
	if err != nil {
		log.Printf("[ERROR] API call failed (endpoint: %s): %v", cfg.API.Endpoint, err)
		return map[string]interface{}{"status": "error", "message": "API call failed", "endpoint": cfg.API.Endpoint}
	}

	log.Printf("[INFO] Successfully sent telemetry data to %s", cfg.API.Endpoint)
	return response
}

func main() {
	// 配置 logger
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("[Claude-Analysis] ")

	// 获取环境变量中的 base URL，如果没有则使用默认值
	defaultBaseURL := "https://gaia.mediatek.inc/o11y/upload_locs"
	if envURL := os.Getenv("O11Y_BASE_URL"); envURL != "" {
		defaultBaseURL = envURL
		log.Printf("[INFO] Read API endpoint from environment variable O11Y_BASE_URL: %s", envURL)
	}

	// Parse command line flags (命令行参数优先级最高)
	var o11yBaseURL = flag.String("o11y_base_url", defaultBaseURL, "Base URL for o11y API endpoint")
	var showVersion = flag.Bool("version", false, "Show version information")
	var checkUpdate = flag.Bool("check-update", false, "Check for available updates")
	var skipUpdateCheck = flag.Bool("skip-update-check", false, "Skip automatic update check")
	var inputPath = flag.String("path", "", "Path to JSONL file to analyze (alternative to stdin mode)")
	var outputPath = flag.String("output", "", "Output path to save analysis result as JSON file (optional)")
	flag.Parse()

	// Handle update-related flags first
	if *checkUpdate {
		log.Printf("[INFO] Checking for updates...")
		result, err := updater.CheckForUpdatesGraceful()
		if err != nil {
			log.Printf("[WARN] Failed to check for updates: %v", err)
			// 創建錯誤結果並輸出，但不退出程序
			errorResult := &updater.UpdateResult{
				CurrentVersion: version.GetVersion(),
				HasUpdate:      false,
				Message:        "Update check failed, but application will continue",
				Error:          err.Error(),
			}
			jsonOutput, _ := json.MarshalIndent(errorResult, "", "  ")
			fmt.Println(string(jsonOutput))
		} else {
			jsonOutput, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(jsonOutput))
		}
		return
	}

	// If version flag is set, print version and exit
	if *showVersion {
		versionInfo := version.Get()
		fmt.Printf("Claude Analysis Tool\n")
		fmt.Printf("Version: %s\n", versionInfo.Version)
		fmt.Printf("Build Time: %s\n", versionInfo.BuildTime)
		fmt.Printf("Git Commit: %s\n", versionInfo.GitCommit)
		fmt.Printf("Go Version: %s\n", versionInfo.GoVersion)
		return
	}

	// Handle path mode (direct JSONL file analysis)
	if *inputPath != "" {
		log.Printf("[INFO] Path mode: analyzing JSONL file %s", *inputPath)
		if err := parseJSONLFile(*inputPath, *outputPath); err != nil {
			log.Printf("[ERROR] Failed to analyze JSONL file: %v", err)
			fmt.Printf(`{"status": "error", "message": "%s"}`, err.Error())
			os.Exit(1)
		}
		return
	}

	// 自動檢查更新（除非用戶明確跳過）
	if !*skipUpdateCheck {
		if err := updater.ForceUpdateCheck(); err != nil {
			log.Printf("[WARN] Update check failed: %v", err)
		}
	}

	// 确定最终使用的 URL
	finalURL := *o11yBaseURL
	if finalURL != defaultBaseURL && os.Getenv("O11Y_BASE_URL") != "" {
		log.Printf("[INFO] Command line argument --o11y_base_url overrides environment variable, using: %s", finalURL)
	}

	log.Printf("[INFO] claude_analysis starting...")
	inputData := readStdinAndSave(finalURL)

	log.Printf("[INFO] readStdinAndSave completed, preparing output...")
	if len(inputData) > 0 {
		jsonOutput, err := json.MarshalIndent(inputData, "", "  ")
		if err != nil {
			log.Printf("[ERROR] JSON marshaling failed: %v", err)
			// 即使序列化失敗，也輸出簡單的錯誤訊息而不是中斷程序
			fmt.Println(`{"status": "error", "message": "JSON marshaling failed"}`)
		} else {
			fmt.Println(string(jsonOutput))
		}
	} else {
		log.Printf("[WARN] No data to output")
		fmt.Println(`{"status": "no_data", "message": "no data to output"}`)
	}
	log.Printf("[INFO] claude_analysis completed")
}

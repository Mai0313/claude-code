package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"claude_analysis/core/config"
	"claude_analysis/core/telemetry"
)

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

	// 讀取 stdin
	stdinData, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Printf("[ERROR] 無法讀取標準輸入: %v", err)
		return map[string]interface{}{"status": "error", "message": "failed to read stdin"}
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
			} else {
				log.Printf("[WARN] JSON 解析失敗，跳過此行: %v", err)
			}
		}
		if len(raw) == 0 {
			log.Printf("[ERROR] POST_TOOL 模式下未找到有效的 JSON 行")
			return map[string]interface{}{"status": "error", "message": "no valid JSON lines found"}
		}
		aggregated = telemetry.AggregateConversationStats(raw)
	} else { // STOP (default)
		path, err := telemetry.ExtractTranscriptPath(string(stdinData))
		if err != nil {
			log.Printf("[ERROR] 無法提取 transcript 路徑: %v", err)
			return map[string]interface{}{"status": "error", "message": "failed to extract transcript path"}
		}
		log.Printf("[INFO] 讀取到的資料路徑: %s", path)
		data, err := telemetry.ReadJSONL(path)
		if err != nil {
			log.Printf("[ERROR] 無法讀取 JSONL 檔案: %v", err)
			return map[string]interface{}{"status": "error", "message": "failed to read JSONL file"}
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
		log.Printf("[ERROR] API 呼叫失敗 (端點: %s): %v", cfg.API.Endpoint, err)
		return map[string]interface{}{"status": "error", "message": "API call failed", "endpoint": cfg.API.Endpoint}
	}

	log.Printf("[INFO] 成功發送遙測資料到 %s", cfg.API.Endpoint)
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
		log.Printf("[INFO] 从环境变量 O11Y_BASE_URL 读取到 API 端点: %s", envURL)
	}

	// Parse command line flags (命令行参数优先级最高)
	var o11yBaseURL = flag.String("o11y_base_url", defaultBaseURL, "Base URL for o11y API endpoint")
	flag.Parse()

	// 确定最终使用的 URL
	finalURL := *o11yBaseURL
	if finalURL != defaultBaseURL && os.Getenv("O11Y_BASE_URL") != "" {
		log.Printf("[INFO] 命令行参数 --o11y_base_url 覆盖了环境变量，使用: %s", finalURL)
	}

	log.Printf("[INFO] claude_analysis 啟動...")
	inputData := readStdinAndSave(finalURL)

	log.Printf("[INFO] readStdinAndSave 執行完成，準備輸出結果...")
	if len(inputData) > 0 {
		jsonOutput, err := json.MarshalIndent(inputData, "", "  ")
		if err != nil {
			log.Printf("[ERROR] JSON 序列化失敗: %v", err)
			// 即使序列化失敗，也輸出簡單的錯誤訊息而不是中斷程序
			fmt.Println(`{"status": "error", "message": "JSON marshaling failed"}`)
		} else {
			fmt.Println(string(jsonOutput))
		}
	} else {
		log.Printf("[WARN] 無資料可輸出")
		fmt.Println(`{"status": "no_data", "message": "no data to output"}`)
	}
	log.Printf("[INFO] claude_analysis 執行完成")
}

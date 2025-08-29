package telemetry

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// pythonDictToJSON 將 Python 字典格式的字串轉換為 JSON 格式
func pythonDictToJSON(pythonDict string) string {
	result := strings.ReplaceAll(pythonDict, "'", "\"")
	result = strings.ReplaceAll(result, "False", "false")
	result = strings.ReplaceAll(result, "True", "true")
	result = strings.ReplaceAll(result, "None", "null")
	return result
}

// ExtractTranscriptPath 從 Python 字典格式的字串中提取 transcript_path
func ExtractTranscriptPath(input string) (string, error) {
	jsonStr := pythonDictToJSON(input)
	jsonBytes := []byte(jsonStr)
	var data map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}
	transcriptPath, exists := data["transcript_path"]
	if !exists {
		return "", fmt.Errorf("找不到 transcript_path")
	}
	pathStr, ok := transcriptPath.(string)
	if !ok {
		return "", fmt.Errorf("transcript_path 不是字串類型")
	}
	return pathStr, nil
}

// ReadJSONL 讀取 JSONL 文件並返回所有 JSON 對象
func ReadJSONL(filename string) ([]map[string]interface{}, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("無法打開文件 %s: %v", filename, err)
	}
	defer file.Close()

	var results []map[string]interface{}
	dec := json.NewDecoder(file)
	index := 0
	for {
		var obj map[string]interface{}
		if err := dec.Decode(&obj); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("解析第 %d 行 JSON 失敗: %v", index+1, err)
		}
		results = append(results, obj)
		index++
	}

	return results, nil
}

package telemetry

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// ClaudeCodeAnalysisDetailBase - 基础详情模型，包含共同的必需字段
type ClaudeCodeAnalysisDetailBase struct {
	FilePath       string `json:"filePath"`
	LineCount      int    `json:"lineCount"`
	CharacterCount int    `json:"characterCount"`
	Timestamp      int64  `json:"timestamp"`
}

// ClaudeCodeAnalysisWriteDetail - writeToFileDetails: 存储完整内容
type ClaudeCodeAnalysisWriteDetail struct {
	ClaudeCodeAnalysisDetailBase
	Content string `json:"content"`
}

// ClaudeCodeAnalysisReadDetail - readFileDetails: 只有必需字段
type ClaudeCodeAnalysisReadDetail struct {
	ClaudeCodeAnalysisDetailBase
}

// ClaudeCodeAnalysisApplyDiffDetail - applyDiffDetails: 保留 old_string/new_string
type ClaudeCodeAnalysisApplyDiffDetail struct {
	ClaudeCodeAnalysisDetailBase
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

// ClaudeCodeAnalysisRunCommandDetail - runCommandDetails: 存储命令和描述
type ClaudeCodeAnalysisRunCommandDetail struct {
	ClaudeCodeAnalysisDetailBase
	Command     string `json:"command"`
	Description string `json:"description"`
}

// ClaudeCodeAnalysisToolCalls - 工具调用次数计数器
type ClaudeCodeAnalysisToolCalls struct {
	Read      int `json:"Read"`
	Write     int `json:"Write"`
	Edit      int `json:"Edit"`
	TodoWrite int `json:"TodoWrite"`
	Bash      int `json:"Bash"`
}

// ClaudeCodeAnalysisRecord - 单个分析会话的汇总统计
type ClaudeCodeAnalysisRecord struct {
	TotalUniqueFiles     int                                  `json:"totalUniqueFiles"`
	TotalWriteLines      int                                  `json:"totalWriteLines"`
	TotalReadCharacters  int                                  `json:"totalReadCharacters"`
	TotalWriteCharacters int                                  `json:"totalWriteCharacters"`
	TotalDiffCharacters  int                                  `json:"totalDiffCharacters"`
	WriteToFileDetails   []ClaudeCodeAnalysisWriteDetail      `json:"writeToFileDetails"`
	ReadFileDetails      []ClaudeCodeAnalysisReadDetail       `json:"readFileDetails"`
	ApplyDiffDetails     []ClaudeCodeAnalysisApplyDiffDetail  `json:"applyDiffDetails"`
	RunCommandDetails    []ClaudeCodeAnalysisRunCommandDetail `json:"runCommandDetails"`
	ToolCallCounts       ClaudeCodeAnalysisToolCalls          `json:"toolCallCounts"`
	TaskID               string                               `json:"taskId"`
	Timestamp            int64                                `json:"timestamp"`
	FolderPath           string                               `json:"folderPath"`
	GitRemoteURL         string                               `json:"gitRemoteUrl"`
}

// ClaudeCodeAnalysis - 顶级分析负载
type ClaudeCodeAnalysis struct {
	User            string                     `json:"user"`
	ExtensionName   string                     `json:"extensionName"`
	InsightsVersion string                     `json:"insightsVersion"`
	MachineID       string                     `json:"machineId"`
	Records         []ClaudeCodeAnalysisRecord `json:"records"`
}

// ClaudeCodeLog - 对应 Python 中的 ClaudeCodeLog 模型
type ClaudeCodeLog struct {
	ParentUUID    *string     `json:"parentUuid"`
	IsSidechain   bool        `json:"isSidechain"`
	UserType      string      `json:"userType"`
	CWD           string      `json:"cwd"`
	SessionID     string      `json:"sessionId"`
	Version       string      `json:"version"`
	GitBranch     string      `json:"gitBranch"`
	Type          string      `json:"type"`
	UUID          string      `json:"uuid"`
	Timestamp     string      `json:"timestamp"`
	Message       interface{} `json:"message"`
	ToolUseResult interface{} `json:"toolUseResult,omitempty"`
}

// parseISOTimestamp 解析 ISO 时间戳为 Unix 秒数
func parseISOTimestamp(ts string) int64 {
	if ts == "" {
		return 0
	}
	// 尝试解析不同的时间格式
	formats := []string{
		"2006-01-02T15:04:05.000Z", // 带毫秒的 UTC
		time.RFC3339Nano,           // RFC3339 带纳秒
		time.RFC3339,               // RFC3339
		"2006-01-02T15:04:05Z",     // 不带毫秒的 UTC
	}

	for _, format := range formats {
		if t, err := time.Parse(format, ts); err == nil {
			return t.Unix()
		}
	}
	return 0
}

// countLines 计算字符串中的行数
func countLines(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Split(s, "\n"))
}

// AnalyzeConversations 分析对话，完全按照 Python 脚本的逻辑
func AnalyzeConversations(records []map[string]interface{}) ClaudeCodeAnalysis {
	// 累积器，用于所有要发出的记录
	writeDetails := []ClaudeCodeAnalysisWriteDetail{}
	readDetails := []ClaudeCodeAnalysisReadDetail{}
	applyDiffDetails := []ClaudeCodeAnalysisApplyDiffDetail{}
	runDetails := []ClaudeCodeAnalysisRunCommandDetail{}

	toolCounts := ClaudeCodeAnalysisToolCalls{}
	uniqueFiles := make(map[string]struct{})

	totalWriteLines := 0
	totalReadCharacters := 0
	totalWriteCharacters := 0
	totalDiffCharacters := 0

	folderPath := ""
	gitRemoteURL := ""
	taskID := ""
	lastTimestamp := int64(0)

	for _, record := range records {
		// 尝试转换为 ClaudeCodeLog 结构
		recordJSON, err := json.Marshal(record)
		if err != nil {
			continue
		}

		var claudeCodeLog ClaudeCodeLog
		if err := json.Unmarshal(recordJSON, &claudeCodeLog); err != nil {
			// 跳过不符合模型的条目（例如 thinking blocks）
			continue
		}

		// 提取基本信息
		if folderPath == "" {
			folderPath = claudeCodeLog.CWD
		}
		taskID = claudeCodeLog.SessionID

		tsInt := parseISOTimestamp(claudeCodeLog.Timestamp)
		if tsInt > lastTimestamp {
			lastTimestamp = tsInt
		}

		// 计算工具调用（助手 tool_use 仅限）
		if claudeCodeLog.Type == "assistant" && claudeCodeLog.Message != nil {
			if messageMap, ok := claudeCodeLog.Message.(map[string]interface{}); ok {
				if contentArray, ok := messageMap["content"].([]interface{}); ok {
					for _, item := range contentArray {
						if itemMap, ok := item.(map[string]interface{}); ok {
							if itemType, ok := itemMap["type"].(string); ok && itemType == "tool_use" {
								if name, ok := itemMap["name"].(string); ok {
									switch name {
									case "Read":
										toolCounts.Read++
									case "Write":
										toolCounts.Write++
									case "Edit":
										toolCounts.Edit++
									case "TodoWrite":
										toolCounts.TodoWrite++
									case "Bash":
										toolCounts.Bash++
										// 记录 runCommandDetails（从输入中，没有文件；使用 cwd 作为 filePath）
										if inputMap, ok := itemMap["input"].(map[string]interface{}); ok {
											command, _ := inputMap["command"].(string)
											description, _ := inputMap["description"].(string)
											runDetails = append(runDetails, ClaudeCodeAnalysisRunCommandDetail{
												ClaudeCodeAnalysisDetailBase: ClaudeCodeAnalysisDetailBase{
													FilePath:       claudeCodeLog.CWD,
													LineCount:      0,
													CharacterCount: len(command),
													Timestamp:      tsInt,
												},
												Command:     command,
												Description: description,
											})
										}
									}
								}
							}
						}
					}
				}
			}
		}

		// 从 toolUseResult 填充各种 *Details
		if claudeCodeLog.ToolUseResult == nil {
			continue
		}

		turMap, ok := claudeCodeLog.ToolUseResult.(map[string]interface{})
		if !ok {
			continue
		}

		// Read result
		if turType, exists := turMap["type"]; exists && turType == "text" {
			if fileMap, ok := turMap["file"].(map[string]interface{}); ok {
				filePath, _ := fileMap["filePath"].(string)
				content, _ := fileMap["content"].(string)
				numLinesFloat, _ := fileMap["numLines"].(float64)
				numLines := int(numLinesFloat)

				readDetails = append(readDetails, ClaudeCodeAnalysisReadDetail{
					ClaudeCodeAnalysisDetailBase: ClaudeCodeAnalysisDetailBase{
						FilePath:       filePath,
						LineCount:      numLines,
						CharacterCount: utf8.RuneCountInString(content),
						Timestamp:      tsInt,
					},
				})
				uniqueFiles[filePath] = struct{}{}
				totalReadCharacters += utf8.RuneCountInString(content)
			}
		}

		// Write (create) result
		if turType, exists := turMap["type"]; exists && turType == "create" {
			filePath, _ := turMap["filePath"].(string)
			content, _ := turMap["content"].(string)
			lineCount := countLines(content)

			writeDetails = append(writeDetails, ClaudeCodeAnalysisWriteDetail{
				ClaudeCodeAnalysisDetailBase: ClaudeCodeAnalysisDetailBase{
					FilePath:       filePath,
					LineCount:      lineCount,
					CharacterCount: utf8.RuneCountInString(content),
					Timestamp:      tsInt,
				},
				Content: content,
			})
			uniqueFiles[filePath] = struct{}{}
			totalWriteLines += lineCount
			totalWriteCharacters += utf8.RuneCountInString(content)
		}

		// Edit result (applyDiff)
		if filePath, ok := turMap["filePath"].(string); ok {
			if newString, ok := turMap["newString"].(string); ok {
				oldString, _ := turMap["oldString"].(string)
				lineCount := countLines(newString)

				applyDiffDetails = append(applyDiffDetails, ClaudeCodeAnalysisApplyDiffDetail{
					ClaudeCodeAnalysisDetailBase: ClaudeCodeAnalysisDetailBase{
						FilePath:       filePath,
						LineCount:      lineCount,
						CharacterCount: utf8.RuneCountInString(newString),
						Timestamp:      tsInt,
					},
					OldString: oldString,
					NewString: newString,
				})
				uniqueFiles[filePath] = struct{}{}
				totalDiffCharacters += utf8.RuneCountInString(newString)
			}
		}
	}

	// 获取 Git remote URL
	gitRemoteURL = getGitRemoteOriginURL(folderPath)

	record := ClaudeCodeAnalysisRecord{
		TotalUniqueFiles:     len(uniqueFiles),
		TotalWriteLines:      totalWriteLines,
		TotalReadCharacters:  totalReadCharacters,
		TotalWriteCharacters: totalWriteCharacters,
		TotalDiffCharacters:  totalDiffCharacters,
		WriteToFileDetails:   writeDetails,
		ReadFileDetails:      readDetails,
		ApplyDiffDetails:     applyDiffDetails,
		RunCommandDetails:    runDetails,
		ToolCallCounts:       toolCounts,
		TaskID:               taskID,
		Timestamp:            lastTimestamp,
		FolderPath:           folderPath,
		GitRemoteURL:         gitRemoteURL,
	}

	// 返回顶级分析对象（注意：这里需要在调用方设置 user, extensionName 等）
	analysis := ClaudeCodeAnalysis{
		Records: []ClaudeCodeAnalysisRecord{record},
	}

	return analysis
}

// AggregateConversationStats 为了向后兼容，保留原有接口但使用新逻辑
func AggregateConversationStats(records []map[string]interface{}) []ClaudeCodeAnalysisRecord {
	analysis := AnalyzeConversations(records)
	return analysis.Records
}

func getGitRemoteOriginURL(cwd string) string {
	if cwd == "" {
		return ""
	}
	cfgPath := filepath.Join(cwd, ".git", "config")
	f, err := os.Open(cfgPath)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	inOrigin := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// section header
			inOrigin = strings.HasPrefix(line, "[remote \"origin\"")
			continue
		}
		if inOrigin && strings.HasPrefix(line, "url = ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "url = "))
		}
	}
	return ""
}

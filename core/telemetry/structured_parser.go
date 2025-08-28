package telemetry

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// StructuredParser provides methods to parse JSONL conversation records into structured data
type StructuredParser struct {
	// Accumulators for all records we will emit
	writeDetails     []ClaudeCodeAnalysisWriteDetail
	readDetails      []ClaudeCodeAnalysisReadDetail
	applyDiffDetails []ClaudeCodeAnalysisApplyDiffDetail
	runDetails       []ClaudeCodeAnalysisRunCommandDetail

	toolCounts ClaudeCodeAnalysisToolCalls
	uniqueFiles map[string]struct{}

	totalWriteLines     int
	totalReadCharacters int
	totalWriteCharacters int
	totalDiffCharacters int

	folderPath    string
	gitRemoteURL  string
	taskID        string
	lastTimestamp int64
}

// NewStructuredParser creates a new structured parser instance
func NewStructuredParser() *StructuredParser {
	return &StructuredParser{
		writeDetails:     make([]ClaudeCodeAnalysisWriteDetail, 0),
		readDetails:      make([]ClaudeCodeAnalysisReadDetail, 0),
		applyDiffDetails: make([]ClaudeCodeAnalysisApplyDiffDetail, 0),
		runDetails:       make([]ClaudeCodeAnalysisRunCommandDetail, 0),
		toolCounts:       ClaudeCodeAnalysisToolCalls{},
		uniqueFiles:      make(map[string]struct{}),
	}
}

// ParseConversationLog parses a single JSONL record into structured data
func (p *StructuredParser) ParseConversationLog(rawData map[string]interface{}) error {
	// Parse the raw data into our structured ClaudeCodeLog
	var logRecord ClaudeCodeLog
	jsonData, err := json.Marshal(rawData)
	if err != nil {
		return fmt.Errorf("failed to marshal raw data: %w", err)
	}
	
	if err := json.Unmarshal(jsonData, &logRecord); err != nil {
		// Skip entries that don't fit the model (e.g., thinking blocks)
		return fmt.Errorf("failed to parse log record: %w", err)
	}

	// Extract context information
	if p.folderPath == "" && logRecord.CWD != "" {
		p.folderPath = logRecord.CWD
	}
	if p.taskID == "" && logRecord.SessionID != "" {
		p.taskID = logRecord.SessionID
	}

	// Parse timestamp
	tsInt := ParseTimestamp(logRecord.Timestamp)
	if tsInt > p.lastTimestamp {
		p.lastTimestamp = tsInt
	}

	// Process assistant messages to count tool invocations
	if logRecord.Type == "assistant" {
		p.processAssistantMessage(logRecord, tsInt)
	}

	// Process tool use results
	if logRecord.ToolUseResult != nil {
		p.processToolUseResult(logRecord, tsInt)
	}

	return nil
}

// processAssistantMessage processes assistant messages to count tool calls
func (p *StructuredParser) processAssistantMessage(logRecord ClaudeCodeLog, timestamp int64) {
	// Try to parse message as assistant message
	messageData, err := json.Marshal(logRecord.Message)
	if err != nil {
		return
	}

	var assistantMsg ClaudeCodeLogAssistantMessage
	if err := json.Unmarshal(messageData, &assistantMsg); err != nil {
		return
	}

	// Process content items to count tool calls
	for _, item := range assistantMsg.Content {
		itemData, err := json.Marshal(item)
		if err != nil {
			continue
		}

		var toolUse ClaudeCodeLogContentToolUse
		if err := json.Unmarshal(itemData, &toolUse); err != nil {
			continue
		}

		if toolUse.Type == "tool_use" {
			switch toolUse.Name {
			case "Read":
				p.toolCounts.Read++
			case "Write":
				p.toolCounts.Write++
			case "Edit":
				p.toolCounts.Edit++
			case "TodoWrite":
				p.toolCounts.TodoWrite++
			case "Bash":
				p.toolCounts.Bash++
				// Record runCommandDetails from the input
				p.processBashInput(toolUse.Input, logRecord.CWD, timestamp)
			}
		}
	}
}

// processBashInput processes bash command input to create run command details
func (p *StructuredParser) processBashInput(input interface{}, cwd string, timestamp int64) {
	inputData, err := json.Marshal(input)
	if err != nil {
		return
	}

	var bashInput ClaudeCodeLogContentInputBash
	if err := json.Unmarshal(inputData, &bashInput); err != nil {
		return
	}

	p.runDetails = append(p.runDetails, ClaudeCodeAnalysisRunCommandDetail{
		ClaudeCodeAnalysisDetailBase: ClaudeCodeAnalysisDetailBase{
			FilePath:       cwd,
			LineCount:      0,
			CharacterCount: utf8.RuneCountInString(bashInput.Command),
			Timestamp:      timestamp,
		},
		Command:     bashInput.Command,
		Description: bashInput.Description,
	})
}

// processToolUseResult processes tool use results to create detail records
func (p *StructuredParser) processToolUseResult(logRecord ClaudeCodeLog, timestamp int64) {
	resultData, err := json.Marshal(logRecord.ToolUseResult)
	if err != nil {
		return
	}

	// Try to determine the type of tool result
	var resultMap map[string]interface{}
	if err := json.Unmarshal(resultData, &resultMap); err != nil {
		return
	}

	// Check for Read result
	if resultType, ok := resultMap["type"].(string); ok && resultType == "text" {
		if fileObj, ok := resultMap["file"].(map[string]interface{}); ok {
			p.processReadResult(fileObj, timestamp)
			return
		}
	}

	// Check for Create result
	if resultType, ok := resultMap["type"].(string); ok && resultType == "create" {
		p.processCreateResult(resultMap, timestamp)
		return
	}

	// Check for Edit result
	if _, hasOldString := resultMap["oldString"]; hasOldString {
		p.processEditResult(resultMap, timestamp)
		return
	}
}

// processReadResult processes a read tool result
func (p *StructuredParser) processReadResult(fileObj map[string]interface{}, timestamp int64) {
	filePath, _ := fileObj["filePath"].(string)
	content, _ := fileObj["content"].(string)
	numLines, _ := fileObj["numLines"].(float64)

	if content == "" {
		return
	}

	p.readDetails = append(p.readDetails, ClaudeCodeAnalysisReadDetail{
		ClaudeCodeAnalysisDetailBase: ClaudeCodeAnalysisDetailBase{
			FilePath:       filePath,
			LineCount:      int(numLines),
			CharacterCount: utf8.RuneCountInString(content),
			Timestamp:      timestamp,
		},
	})

	if filePath != "" {
		p.uniqueFiles[filePath] = struct{}{}
	}
	p.totalReadCharacters += utf8.RuneCountInString(content)
}

// processCreateResult processes a create tool result
func (p *StructuredParser) processCreateResult(resultMap map[string]interface{}, timestamp int64) {
	filePath, _ := resultMap["filePath"].(string)
	content, _ := resultMap["content"].(string)

	if content == "" {
		return
	}

	lineCount := CountLines(content)
	characterCount := utf8.RuneCountInString(content)

	p.writeDetails = append(p.writeDetails, ClaudeCodeAnalysisWriteDetail{
		ClaudeCodeAnalysisDetailBase: ClaudeCodeAnalysisDetailBase{
			FilePath:       filePath,
			LineCount:      lineCount,
			CharacterCount: characterCount,
			Timestamp:      timestamp,
		},
		Content: content,
	})

	if filePath != "" {
		p.uniqueFiles[filePath] = struct{}{}
	}
	p.totalWriteLines += lineCount
	p.totalWriteCharacters += characterCount
}

// processEditResult processes an edit tool result
func (p *StructuredParser) processEditResult(resultMap map[string]interface{}, timestamp int64) {
	filePath, _ := resultMap["filePath"].(string)
	newString, _ := resultMap["newString"].(string)
	oldString, _ := resultMap["oldString"].(string)

	if newString == "" {
		return
	}

	lineCount := CountLines(newString)
	characterCount := utf8.RuneCountInString(newString)

	p.applyDiffDetails = append(p.applyDiffDetails, ClaudeCodeAnalysisApplyDiffDetail{
		ClaudeCodeAnalysisDetailBase: ClaudeCodeAnalysisDetailBase{
			FilePath:       filePath,
			LineCount:      lineCount,
			CharacterCount: characterCount,
			Timestamp:      timestamp,
		},
		OldString: oldString,
		NewString: newString,
	})

	if filePath != "" {
		p.uniqueFiles[filePath] = struct{}{}
	}
	p.totalDiffCharacters += characterCount
}

// GetAnalysisRecord returns the aggregated analysis record
func (p *StructuredParser) GetAnalysisRecord() ClaudeCodeAnalysisRecord {
	// Get git remote URL if folder path is available
	if p.gitRemoteURL == "" && p.folderPath != "" {
		p.gitRemoteURL = getGitRemoteOriginURL(p.folderPath)
	}

	return ClaudeCodeAnalysisRecord{
		TotalUniqueFiles:     len(p.uniqueFiles),
		TotalWriteLines:      p.totalWriteLines,
		TotalReadCharacters:  p.totalReadCharacters,
		TotalWriteCharacters: p.totalWriteCharacters,
		TotalDiffCharacters:  p.totalDiffCharacters,
		WriteToFileDetails:   p.writeDetails,
		ReadFileDetails:      p.readDetails,
		ApplyDiffDetails:     p.applyDiffDetails,
		RunCommandDetails:    p.runDetails,
		ToolCallCounts:       p.toolCounts,
		TaskID:               p.taskID,
		Timestamp:            p.lastTimestamp,
		FolderPath:           p.folderPath,
		GitRemoteURL:         p.gitRemoteURL,
	}
}

// AnalyzeConversations is the main function that processes JSONL records and returns analysis
func AnalyzeConversations(records []map[string]interface{}) []ClaudeCodeAnalysisRecord {
	if len(records) == 0 {
		return []ClaudeCodeAnalysisRecord{}
	}

	parser := NewStructuredParser()

	// Process each record
	for _, record := range records {
		if err := parser.ParseConversationLog(record); err != nil {
			// Log error but continue processing other records
			continue
		}
	}

	// Return the aggregated record
	return []ClaudeCodeAnalysisRecord{parser.GetAnalysisRecord()}
}

// getGitRemoteOriginURL attempts to read .git/config under cwd and extract remote.origin.url
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
package telemetry

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// AggregateConversationStats transforms raw JSONL event maps into structured analysis records.
// This function uses structured parsing to handle each JSON line with proper type safety.
func AggregateConversationStats(records []map[string]interface{}) []ClaudeCodeAnalysisRecord {
	if len(records) == 0 {
		return []ClaudeCodeAnalysisRecord{}
	}

	log.Printf("[DEBUG] Starting to process %d JSONL records", len(records))

	// Parse each raw record into structured format
	parsedLogs := make([]*ClaudeCodeLog, 0, len(records))
	for i, record := range records {
		parsedLog, err := parseClaudeCodeLog(record)
		if err != nil {
			log.Printf("[WARN] Failed to parse record %d: %v", i+1, err)
			continue // Skip invalid records
		}
		parsedLogs = append(parsedLogs, parsedLog)
	}

	log.Printf("[DEBUG] Successfully parsed %d/%d records", len(parsedLogs), len(records))

	// Now process the structured data
	return processStructuredLogs(parsedLogs)
}

// parseClaudeCodeLog converts a raw map to structured ClaudeCodeLog
func parseClaudeCodeLog(record map[string]interface{}) (*ClaudeCodeLog, error) {
	// Convert map to JSON and back to struct for proper parsing
	jsonBytes, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %w", err)
	}

	var log ClaudeCodeLog
	if err := json.Unmarshal(jsonBytes, &log); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to ClaudeCodeLog: %w", err)
	}

	return &log, nil
}

// processStructuredLogs processes structured logs to generate analysis records
func processStructuredLogs(logs []*ClaudeCodeLog) []ClaudeCodeAnalysisRecord {
	// Accumulators for analysis data
	var (
		writeDetails     []ClaudeCodeAnalysisWriteDetail
		readDetails      []ClaudeCodeAnalysisReadDetail
		applyDiffDetails []ClaudeCodeAnalysisApplyDiffDetail
		runDetails       []ClaudeCodeAnalysisRunCommandDetail
		toolCounts       ClaudeCodeAnalysisToolCalls
		uniqueFiles      = make(map[string]struct{})
		
		totalWriteLines      = 0
		totalReadCharacters  = 0
		totalWriteCharacters = 0
		totalDiffCharacters  = 0
		
		folderPath   = ""
		gitRemoteURL = ""
		taskID       = ""
		lastTimestamp int64 = 0
	)

	// Map to track UUID -> tool name for matching tool results
	uuidToToolName := make(map[string]string)

	// First pass: extract context and count tool calls
	for _, logEntry := range logs {
		// Extract context information
		if folderPath == "" && logEntry.Cwd != "" {
			folderPath = logEntry.Cwd
		}
		if taskID == "" && logEntry.SessionID != "" {
			taskID = logEntry.SessionID
		}

		// Parse timestamp
		if ts := ParseTimestamp(logEntry.Timestamp); ts > lastTimestamp {
			lastTimestamp = ts
		}

		// Process assistant messages for tool calls
		if logEntry.Type == "assistant" {
			processAssistantMessage(logEntry, &toolCounts, &runDetails, uuidToToolName)
		}
	}

	// Second pass: process tool results
	for _, logEntry := range logs {
		if len(logEntry.ToolUseResult) > 0 {
			processToolResult(logEntry, uuidToToolName, &writeDetails, &readDetails, 
							&applyDiffDetails, uniqueFiles, &totalWriteLines, 
							&totalReadCharacters, &totalWriteCharacters, &totalDiffCharacters)
		}
	}

	// Get git remote URL
	if folderPath != "" {
		gitRemoteURL = getGitRemoteOriginURL(folderPath)
	}

	// Create the analysis record
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

	log.Printf("[DEBUG] Generated analysis record: %d unique files, %d tool calls", 
		len(uniqueFiles), 
		toolCounts.Read+toolCounts.Write+toolCounts.Edit+toolCounts.TodoWrite+toolCounts.Bash)

	return []ClaudeCodeAnalysisRecord{record}
}

// processAssistantMessage processes assistant messages to extract tool calls and run commands
func processAssistantMessage(logEntry *ClaudeCodeLog, toolCounts *ClaudeCodeAnalysisToolCalls, 
							runDetails *[]ClaudeCodeAnalysisRunCommandDetail, uuidToToolName map[string]string) {
	
	var assistantMsg ClaudeCodeLogAssistantMessage
	if err := json.Unmarshal(logEntry.Message, &assistantMsg); err != nil {
		log.Printf("[WARN] Failed to parse assistant message: %v", err)
		return
	}

	timestamp := ParseTimestamp(logEntry.Timestamp)

	// Process content for tool_use
	for _, content := range assistantMsg.Content {
		if content.Type == "tool_use" {
			toolName := content.Name
			uuidToToolName[logEntry.UUID] = toolName

			// Count tool calls
			switch toolName {
			case "read", "Read":
				toolCounts.Read++
			case "write", "Write":
				toolCounts.Write++
			case "edit", "Edit":
				toolCounts.Edit++
			case "todo_write", "TodoWrite":
				toolCounts.TodoWrite++
			case "bash", "Bash":
				toolCounts.Bash++
				// Process Bash input for run command details
				processBashInput(content.Input, logEntry.Cwd, timestamp, runDetails)
			}
		}
	}
}

// processBashInput extracts command details from Bash tool input
func processBashInput(inputRaw json.RawMessage, cwd string, timestamp int64, runDetails *[]ClaudeCodeAnalysisRunCommandDetail) {
	var bashInput ClaudeCodeLogContentInputBash
	if err := json.Unmarshal(inputRaw, &bashInput); err != nil {
		log.Printf("[WARN] Failed to parse bash input: %v", err)
		return
	}

	if bashInput.Command != "" {
		*runDetails = append(*runDetails, ClaudeCodeAnalysisRunCommandDetail{
			ClaudeCodeAnalysisDetailBase: ClaudeCodeAnalysisDetailBase{
				FilePath:       cwd, // Use cwd as filePath for commands
				LineCount:      0,
				CharacterCount: len(bashInput.Command),
				Timestamp:      timestamp,
			},
			Command:     bashInput.Command,
			Description: bashInput.Description,
		})
	}
}

// processToolResult processes tool use results to extract file operation details
func processToolResult(logEntry *ClaudeCodeLog, uuidToToolName map[string]string,
					  writeDetails *[]ClaudeCodeAnalysisWriteDetail,
					  readDetails *[]ClaudeCodeAnalysisReadDetail,
					  applyDiffDetails *[]ClaudeCodeAnalysisApplyDiffDetail,
					  uniqueFiles map[string]struct{},
					  totalWriteLines, totalReadChars, totalWriteChars, totalDiffChars *int) {

	parentUUID := ""
	if logEntry.ParentUUID != nil {
		parentUUID = *logEntry.ParentUUID
	}
	toolName := uuidToToolName[parentUUID]
	timestamp := ParseTimestamp(logEntry.Timestamp)

	// Try to parse different types of tool results
	switch strings.ToLower(toolName) {
	case "read":
		processReadResult(logEntry.ToolUseResult, timestamp, readDetails, uniqueFiles, totalReadChars)
	case "write":
		processWriteResult(logEntry.ToolUseResult, timestamp, writeDetails, uniqueFiles, totalWriteLines, totalWriteChars)
	case "edit":
		processEditResult(logEntry.ToolUseResult, timestamp, applyDiffDetails, uniqueFiles, totalDiffChars)
	}
}

// processReadResult processes Read tool results
func processReadResult(resultRaw json.RawMessage, timestamp int64,
					  readDetails *[]ClaudeCodeAnalysisReadDetail,
					  uniqueFiles map[string]struct{}, totalReadChars *int) {
	
	var readResult ClaudeCodeLogToolUseResultRead
	if err := json.Unmarshal(resultRaw, &readResult); err != nil {
		log.Printf("[WARN] Failed to parse read result: %v", err)
		return
	}

	if readResult.File.FilePath != "" {
		uniqueFiles[readResult.File.FilePath] = struct{}{}
		chars := utf8.RuneCountInString(readResult.File.Content)
		
		*readDetails = append(*readDetails, ClaudeCodeAnalysisReadDetail{
			ClaudeCodeAnalysisDetailBase: ClaudeCodeAnalysisDetailBase{
				FilePath:       readResult.File.FilePath,
				LineCount:      readResult.File.NumLines,
				CharacterCount: chars,
				Timestamp:      timestamp,
			},
		})
		*totalReadChars += chars
	}
}

// processWriteResult processes Write tool results  
func processWriteResult(resultRaw json.RawMessage, timestamp int64,
					   writeDetails *[]ClaudeCodeAnalysisWriteDetail,
					   uniqueFiles map[string]struct{}, 
					   totalWriteLines, totalWriteChars *int) {
	
	var writeResult ClaudeCodeLogToolUseResultCreate
	if err := json.Unmarshal(resultRaw, &writeResult); err != nil {
		log.Printf("[WARN] Failed to parse write result: %v", err)
		return
	}

	if writeResult.FilePath != "" {
		uniqueFiles[writeResult.FilePath] = struct{}{}
		chars := utf8.RuneCountInString(writeResult.Content)
		lines := CountLines(writeResult.Content)
		
		*writeDetails = append(*writeDetails, ClaudeCodeAnalysisWriteDetail{
			ClaudeCodeAnalysisDetailBase: ClaudeCodeAnalysisDetailBase{
				FilePath:       writeResult.FilePath,
				LineCount:      lines,
				CharacterCount: chars,
				Timestamp:      timestamp,
			},
			Content: writeResult.Content,
		})
		*totalWriteChars += chars
		*totalWriteLines += lines
	}
}

// processEditResult processes Edit tool results
func processEditResult(resultRaw json.RawMessage, timestamp int64,
					  applyDiffDetails *[]ClaudeCodeAnalysisApplyDiffDetail,
					  uniqueFiles map[string]struct{}, totalDiffChars *int) {
	
	var editResult ClaudeCodeLogToolUseResultEdit
	if err := json.Unmarshal(resultRaw, &editResult); err != nil {
		log.Printf("[WARN] Failed to parse edit result: %v", err)
		return
	}

	if editResult.FilePath != "" {
		uniqueFiles[editResult.FilePath] = struct{}{}
		chars := utf8.RuneCountInString(editResult.NewString)
		lines := CountLines(editResult.NewString)
		
		*applyDiffDetails = append(*applyDiffDetails, ClaudeCodeAnalysisApplyDiffDetail{
			ClaudeCodeAnalysisDetailBase: ClaudeCodeAnalysisDetailBase{
				FilePath:       editResult.FilePath,
				LineCount:      lines,
				CharacterCount: chars,
				Timestamp:      timestamp,
			},
			OldString: editResult.OldString,
			NewString: editResult.NewString,
		})
		*totalDiffChars += chars
	}
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

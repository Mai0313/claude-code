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

// WriteToFileDetail mirrors the target schema for write operations
type WriteToFileDetail struct {
	FilePath        string `json:"filePath"`
	LineCount       int    `json:"lineCount"`
	CharacterCount  int    `json:"characterCount"`
	Timestamp       int64  `json:"timestamp"`
	AiOutputContent string `json:"aiOutputContent"`
	FileContent     string `json:"fileContent"`
}

// ReadFileDetail mirrors the target schema for read operations
type ReadFileDetail struct {
	FilePath        string `json:"filePath"`
	CharacterCount  int    `json:"characterCount"`
	Timestamp       int64  `json:"timestamp"`
	AiOutputContent string `json:"aiOutputContent"`
	FileContent     string `json:"fileContent"`
}

// ApplyDiffDetail mirrors the target schema for apply-diff operations
type ApplyDiffDetail struct {
	FilePath        string `json:"filePath"`
	CharacterCount  int    `json:"characterCount"`
	Timestamp       int64  `json:"timestamp"`
	AiOutputContent string `json:"aiOutputContent"`
	FileContent     string `json:"fileContent"`
}

// ApiConversationStats is the aggregated record we will attach to payload.records
type ApiConversationStats struct {
	TotalUniqueFiles     int                 `json:"totalUniqueFiles"`
	TotalWriteLines      int                 `json:"totalWriteLines"`
	TotalReadCharacters  int                 `json:"totalReadCharacters"`
	TotalWriteCharacters int                 `json:"totalWriteCharacters"`
	TotalDiffCharacters  int                 `json:"totalDiffCharacters"`
	WriteToFileDetails   []WriteToFileDetail `json:"writeToFileDetails"`
	ReadFileDetails      []ReadFileDetail    `json:"readFileDetails"`
	ApplyDiffDetails     []ApplyDiffDetail   `json:"applyDiffDetails"`
	ToolCallCounts       map[string]int      `json:"toolCallCounts"`
	TaskID               string              `json:"taskId"`
	Timestamp            int64               `json:"timestamp"`
	FolderPath           string              `json:"folderPath"`
	GitRemoteURL         string              `json:"gitRemoteUrl"`
}

// AggregateConversationStats transforms raw JSONL event maps into a single aggregated stats record.
// It is designed to be called right after ReadJSONL without changing core logic elsewhere.
func AggregateConversationStats(records []map[string]interface{}) []ApiConversationStats {
	if len(records) == 0 {
		return []ApiConversationStats{}
	}

	// Normalize each generic record into a strongly typed struct first
	typed := make([]*ClaudeCodeLog, 0, len(records))
	for _, rec := range records {
		cl, err := normalizeRecord(rec)
		if err != nil {
			// skip malformed lines rather than failing the whole aggregation
			continue
		}
		typed = append(typed, cl)
	}

	// Map assistant message UUID -> tool name (e.g., "Read", "Write")
	uuidToToolName := make(map[string]string)
	toolCallCounts := make(map[string]int)

	var (
		cwd       string
		sessionID string
		lastMs    int64
	)

	// First pass: extract context (cwd, sessionId), collect tool calls
	for _, rec := range typed {
		if rec == nil {
			continue
		}
		if v := rec.Cwd; v != "" && cwd == "" {
			cwd = v
		}
		if v := rec.SessionID; v != "" && sessionID == "" {
			sessionID = v
		}
		if ms := parseISOMillis(rec.Timestamp); ms > lastMs {
			lastMs = ms
		}

		// Parse assistant tool_use events
		if rec.Type == "assistant" {
			uuid := rec.UUID
			for _, item := range rec.Message.Content {
				if item.Type == "tool_use" && item.Name != "" {
					uuidToToolName[uuid] = item.Name
					toolCallCounts[item.Name]++
				}
			}
		}
	}

	// Second pass: collect tool results and build details
	writeDetails := make([]WriteToFileDetail, 0)
	readDetails := make([]ReadFileDetail, 0)
	applyDiffDetails := make([]ApplyDiffDetail, 0)
	uniqueFiles := make(map[string]struct{})
	var totalWriteLines int
	var totalReadChars int
	var totalWriteChars int
	var totalDiffChars int

	for _, rec := range typed {
		if rec == nil || rec.ToolUseResult == nil {
			continue
		}
		parentUUID := rec.ParentUUID
		toolName := uuidToToolName[parentUUID]

		// Extract filePath/content from either nested file object or top-level fields
		var filePath string
		var content string
		if rec.ToolUseResult.File != nil {
			if v := rec.ToolUseResult.File.FilePath; v != "" {
				filePath = v
			}
			if v := rec.ToolUseResult.File.Content; v != "" {
				content = v
			}
		}
		if filePath == "" {
			if v := rec.ToolUseResult.FilePath; v != "" {
				filePath = v
			}
		}
		if content == "" {
			if v := rec.ToolUseResult.Content; v != "" {
				content = v
			}
		}
		// Fallback: if still no content but there is a structuredPatch, serialize it
		if content == "" && rec.ToolUseResult.StructuredPatch != nil {
			if b, err := json.Marshal(rec.ToolUseResult.StructuredPatch); err == nil {
				content = string(b)
			}
		}

		tsMs := parseISOMillis(rec.Timestamp)

		if filePath != "" {
			uniqueFiles[filePath] = struct{}{}
		}

		switch strings.ToLower(toolName) {
		case "read":
			if content == "" {
				continue
			}
			chars := utf8.RuneCountInString(content)
			readDetails = append(readDetails, ReadFileDetail{
				FilePath:        filePath,
				CharacterCount:  chars,
				Timestamp:       tsMs,
				AiOutputContent: "",
				FileContent:     content,
			})
			totalReadChars += chars
		case "write":
			if content == "" {
				continue
			}
			chars := utf8.RuneCountInString(content)
			lines := countLines(content)
			writeDetails = append(writeDetails, WriteToFileDetail{
				FilePath:        filePath,
				LineCount:       lines,
				CharacterCount:  chars,
				Timestamp:       tsMs,
				AiOutputContent: content,
				FileContent:     content,
			})
			totalWriteChars += chars
			totalWriteLines += lines
		case "applydiff", "apply_diff", "applypatch":
			if content == "" {
				continue
			}
			chars := utf8.RuneCountInString(content)
			applyDiffDetails = append(applyDiffDetails, ApplyDiffDetail{
				FilePath:        filePath,
				CharacterCount:  chars,
				Timestamp:       tsMs,
				AiOutputContent: content,
				FileContent:     content,
			})
			totalDiffChars += chars
		default:
			// Unknown tool; ignore for details but still counted in ToolCallCounts above
		}
	}

	gitURL := getGitRemoteOriginURL(cwd)

	stats := ApiConversationStats{
		TotalUniqueFiles:     len(uniqueFiles),
		TotalWriteLines:      totalWriteLines,
		TotalReadCharacters:  totalReadChars,
		TotalWriteCharacters: totalWriteChars,
		TotalDiffCharacters:  totalDiffChars,
		WriteToFileDetails:   writeDetails,
		ReadFileDetails:      readDetails,
		ApplyDiffDetails:     applyDiffDetails,
		ToolCallCounts:       toolCallCounts,
		TaskID:               sessionID,
		Timestamp:            lastMs,
		FolderPath:           cwd,
		GitRemoteURL:         gitURL,
	}

	return []ApiConversationStats{stats}
}

func parseISOMillis(iso string) int64 {
	if iso == "" {
		return 0
	}
	// Try RFC3339 with or without fractional seconds
	if t, err := time.Parse(time.RFC3339Nano, iso); err == nil {
		return t.UnixNano() / int64(time.Millisecond)
	}
	if t, err := time.Parse(time.RFC3339, iso); err == nil {
		return t.UnixNano() / int64(time.Millisecond)
	}
	return 0
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	// Count lines by splitting; handle trailing newline correctly
	lines := 1
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines++
		}
	}
	return lines
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

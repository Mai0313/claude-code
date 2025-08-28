package telemetry

import (
	"time"
)

// ============================================================================
// Claude Code Analysis Models - data used for analysis stats
// ============================================================================

// ClaudeCodeAnalysisDetailBase represents the base detail model with shared required fields
type ClaudeCodeAnalysisDetailBase struct {
	FilePath       string `json:"filePath"`
	LineCount      int    `json:"lineCount"`
	CharacterCount int    `json:"characterCount"`
	Timestamp      int64  `json:"timestamp"`
}

// ClaudeCodeAnalysisWriteDetail represents writeToFileDetails with full content
type ClaudeCodeAnalysisWriteDetail struct {
	ClaudeCodeAnalysisDetailBase
	Content string `json:"content"`
}

// ClaudeCodeAnalysisReadDetail represents readFileDetails
type ClaudeCodeAnalysisReadDetail struct {
	ClaudeCodeAnalysisDetailBase
}

// ClaudeCodeAnalysisApplyDiffDetail represents applyDiffDetails with old/new strings
type ClaudeCodeAnalysisApplyDiffDetail struct {
	ClaudeCodeAnalysisDetailBase
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

// ClaudeCodeAnalysisRunCommandDetail represents runCommandDetails
type ClaudeCodeAnalysisRunCommandDetail struct {
	ClaudeCodeAnalysisDetailBase
	Command     string `json:"command"`
	Description string `json:"description"`
}

// ClaudeCodeAnalysisToolCalls represents counters for tool invocations
type ClaudeCodeAnalysisToolCalls struct {
	Read      int `json:"Read"`
	Write     int `json:"Write"`
	Edit      int `json:"Edit"`
	TodoWrite int `json:"TodoWrite"`
	Bash      int `json:"Bash"`
}

// ClaudeCodeAnalysisRecord represents aggregated stats for a single analysis session
type ClaudeCodeAnalysisRecord struct {
	TotalUniqueFiles     int                                `json:"totalUniqueFiles"`
	TotalWriteLines      int                                `json:"totalWriteLines"`
	TotalReadCharacters  int                                `json:"totalReadCharacters"`
	TotalWriteCharacters int                                `json:"totalWriteCharacters"`
	TotalDiffCharacters  int                                `json:"totalDiffCharacters"`
	WriteToFileDetails   []ClaudeCodeAnalysisWriteDetail   `json:"writeToFileDetails"`
	ReadFileDetails      []ClaudeCodeAnalysisReadDetail    `json:"readFileDetails"`
	ApplyDiffDetails     []ClaudeCodeAnalysisApplyDiffDetail `json:"applyDiffDetails"`
	RunCommandDetails    []ClaudeCodeAnalysisRunCommandDetail `json:"runCommandDetails"`
	ToolCallCounts       ClaudeCodeAnalysisToolCalls       `json:"toolCallCounts"`
	TaskID               string                             `json:"taskId"`
	Timestamp            int64                              `json:"timestamp"`
	FolderPath           string                             `json:"folderPath"`
	GitRemoteURL         string                             `json:"gitRemoteUrl"`
}

// ClaudeCodeAnalysis represents the top-level analysis payload
type ClaudeCodeAnalysis struct {
	User            string                    `json:"user"`
	ExtensionName   string                    `json:"extensionName"`
	InsightsVersion string                    `json:"insightsVersion"`
	MachineID       string                    `json:"machineId"`
	Records         []ClaudeCodeAnalysisRecord `json:"records"`
}

// ============================================================================
// Claude Code Log Models - parse JSONL conversation records
// ============================================================================

// ClaudeCodeLogContentInputBash represents bash command input
type ClaudeCodeLogContentInputBash struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// ClaudeCodeLogContentInputEdit represents edit operation input
type ClaudeCodeLogContentInputEdit struct {
	FilePath  string `json:"file_path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

// ClaudeCodeLogContentInputRead represents read operation input
type ClaudeCodeLogContentInputRead struct {
	FilePath string `json:"file_path"`
}

// ClaudeCodeLogContentInputTodoWriteItem represents a todo item
type ClaudeCodeLogContentInputTodoWriteItem struct {
	Content    string `json:"content"`
	Status     string `json:"status"`
	ActiveForm string `json:"activeForm"`
}

// ClaudeCodeLogContentInputTodoWrite represents todo write input
type ClaudeCodeLogContentInputTodoWrite struct {
	Todos []ClaudeCodeLogContentInputTodoWriteItem `json:"todos"`
}

// ClaudeCodeLogContentInputWrite represents write operation input
type ClaudeCodeLogContentInputWrite struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// ClaudeCodeLogContentToolUse represents a tool use event
type ClaudeCodeLogContentToolUse struct {
	Type  string      `json:"type"`
	Name  string      `json:"name"`
	ID    string      `json:"id"`
	Input interface{} `json:"input"`
}

// ClaudeCodeLogContentToolResult represents a tool result
type ClaudeCodeLogContentToolResult struct {
	Type        string `json:"type"`
	ToolUseID   string `json:"tool_use_id"`
	Content     string `json:"content"`
}

// ClaudeCodeLogContentText represents text content
type ClaudeCodeLogContentText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ClaudeCodeLogMessageUsage represents message usage statistics
type ClaudeCodeLogMessageUsage struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	OutputTokens             int `json:"output_tokens"`
}

// ============================================================================
// Tool Use Result Models - outputs returned by tools
// ============================================================================

// ClaudeCodeLogToolUseResultFile represents file information in tool results
type ClaudeCodeLogToolUseResultFile struct {
	FilePath   string `json:"filePath"`
	Content    string `json:"content"`
	NumLines   int    `json:"numLines"`
	StartLine  int    `json:"startLine"`
	TotalLines int    `json:"totalLines"`
}

// ClaudeCodeLogToolUseResultRead represents read tool result
type ClaudeCodeLogToolUseResultRead struct {
	Type string                        `json:"type"`
	File ClaudeCodeLogToolUseResultFile `json:"file"`
}

// ClaudeCodeLogToolUseResultCreate represents create tool result
type ClaudeCodeLogToolUseResultCreate struct {
	Type            string        `json:"type"`
	FilePath        string        `json:"filePath"`
	Content         string        `json:"content"`
	StructuredPatch []interface{} `json:"structuredPatch"`
}

// ClaudeCodeLogToolUseResultEditPatch represents edit patch information
type ClaudeCodeLogToolUseResultEditPatch struct {
	OldStart int      `json:"oldStart"`
	OldLines int      `json:"oldLines"`
	NewStart int      `json:"newStart"`
	NewLines int      `json:"newLines"`
	Lines    []string `json:"lines"`
}

// ClaudeCodeLogToolUseResultEdit represents edit tool result
type ClaudeCodeLogToolUseResultEdit struct {
	FilePath        string                                `json:"filePath"`
	OldString       string                                `json:"oldString"`
	NewString       string                                `json:"newString"`
	OriginalFile    string                                `json:"originalFile"`
	StructuredPatch []ClaudeCodeLogToolUseResultEditPatch `json:"structuredPatch"`
	UserModified    bool                                  `json:"userModified"`
	ReplaceAll      bool                                  `json:"replaceAll"`
}

// ClaudeCodeLogToolUseResultBash represents bash tool result
type ClaudeCodeLogToolUseResultBash struct {
	Stdout      string `json:"stdout"`
	Stderr      string `json:"stderr"`
	Interrupted bool   `json:"interrupted"`
	IsImage     bool   `json:"isImage"`
}

// ClaudeCodeLogToolUseResultTodo represents todo tool result
type ClaudeCodeLogToolUseResultTodo struct {
	OldTodos []ClaudeCodeLogContentInputTodoWriteItem `json:"oldTodos"`
	NewTodos []ClaudeCodeLogContentInputTodoWriteItem `json:"newTodos"`
}

// ClaudeCodeLogAssistantMessage represents an assistant message
type ClaudeCodeLogAssistantMessage struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	Role         string        `json:"role"`
	Model        string        `json:"model"`
	Content      []interface{} `json:"content"`
	StopReason   *string       `json:"stop_reason"`
	StopSequence *string       `json:"stop_sequence"`
	Usage        ClaudeCodeLogMessageUsage `json:"usage"`
}

// ClaudeCodeLogUserMessage represents a user message
type ClaudeCodeLogUserMessage struct {
	Role    string        `json:"role"`
	Content interface{}   `json:"content"`
}

// ClaudeCodeLog represents a single JSONL record
type ClaudeCodeLog struct {
	ParentUUID    *string      `json:"parentUuid"`
	IsSidechain   bool         `json:"isSidechain"`
	UserType      string       `json:"userType"`
	CWD           string       `json:"cwd"`
	SessionID     string       `json:"sessionId"`
	Version       string       `json:"version"`
	GitBranch     string       `json:"gitBranch"`
	Type          string       `json:"type"`
	UUID          string       `json:"uuid"`
	Timestamp     string       `json:"timestamp"`
	Message       interface{}  `json:"message"`
	ToolUseResult interface{}  `json:"toolUseResult"`
}

// ParseTimestamp converts ISO timestamp string to Unix milliseconds
func ParseTimestamp(iso string) int64 {
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

// CountLines counts the number of lines in a string
func CountLines(s string) int {
	if s == "" {
		return 0
	}
	lines := 1
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines++
		}
	}
	return lines
}
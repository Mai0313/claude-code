package telemetry

import (
	"encoding/json"
)

// ClaudeContentItem represents one item in assistant message content array.
// We only keep fields we actually use for parsing/aggregation.
type ClaudeContentItem struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// ClaudeMessage holds assistant/user message payload. We only need Content for assistant.
type ClaudeMessage struct {
	Content []ClaudeContentItem `json:"content"`
}

// ClaudeToolUseResultFile mirrors the nested file object used by some tool results.
type ClaudeToolUseResultFile struct {
	FilePath   string `json:"filePath"`
	Content    string `json:"content"`
	NumLines   int    `json:"numLines"`
	StartLine  int    `json:"startLine"`
	TotalLines int    `json:"totalLines"`
}

// ClaudeToolUseResult is a normalized shape that captures the superset of fields
// we care about across different tool result variants (read/write/edit/bash/...).
// This keeps per-line data structured and convenient for Go callers.
type ClaudeToolUseResult struct {
	Type            string                   `json:"type"`
	ToolUseID       string                   `json:"tool_use_id"`
	File            *ClaudeToolUseResultFile `json:"file"`
	FilePath        string                   `json:"filePath"`
	Content         string                   `json:"content"`
	StructuredPatch any                      `json:"structuredPatch"`
	OldString       string                   `json:"oldString"`
	NewString       string                   `json:"newString"`
	Stdout          string                   `json:"stdout"`
	Stderr          string                   `json:"stderr"`
}

// ClaudeCodeLog is the typed representation of each JSONL line.
// Unknown fields in the incoming JSON are ignored.
type ClaudeCodeLog struct {
	ParentUUID    string               `json:"parentUuid"`
	IsSidechain   bool                 `json:"isSidechain"`
	UserType      string               `json:"userType"`
	Cwd           string               `json:"cwd"`
	SessionID     string               `json:"sessionId"`
	Version       string               `json:"version"`
	GitBranch     string               `json:"gitBranch"`
	Type          string               `json:"type"`
	UUID          string               `json:"uuid"`
	Timestamp     string               `json:"timestamp"`
	Message       ClaudeMessage        `json:"message"`
	ToolUseResult *ClaudeToolUseResult `json:"toolUseResult"`
}

// normalizeRecord marshals the generic map into JSON and unmarshals into a typed struct.
// It returns a structured ClaudeCodeLog which is convenient for downstream usage.
func normalizeRecord(rec map[string]interface{}) (*ClaudeCodeLog, error) {
	b, err := json.Marshal(rec)
	if err != nil {
		return nil, err
	}
	var out ClaudeCodeLog
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}


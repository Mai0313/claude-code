package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"claude_analysis/core/config"
	"claude_analysis/core/telemetry"
)

func TestParser_FromTestConversationJSONL_PrintsFullPayload(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to get caller info")
	}
	jsonlPath := filepath.Join(filepath.Dir(filepath.Dir(thisFile)), "examples", "test_conversation.jsonl")

	records, err := telemetry.ReadJSONL(jsonlPath)
	if err != nil {
		t.Fatalf("ReadJSONL error: %v", err)
	}

	analysis := telemetry.AnalyzeConversations(records)
	if len(analysis.Records) != 1 {
		t.Fatalf("expected 1 analysis record, got %d", len(analysis.Records))
	}

	cfg := config.Default()
	analysis.User = cfg.UserName
	analysis.ExtensionName = cfg.ExtensionName
	analysis.MachineID = cfg.MachineID
	analysis.InsightsVersion = cfg.InsightsVersion

	payload := map[string]interface{}{
		"user":            analysis.User,
		"records":         analysis.Records,
		"extensionName":   analysis.ExtensionName,
		"machineId":       analysis.MachineID,
		"insightsVersion": analysis.InsightsVersion,
	}
	pretty, err := json.MarshalIndent(payload, "", "  ")
	if err == nil {
		t.Logf("Full transformed payload:\n%s", string(pretty))
	} else {
		t.Logf("Full transformed payload (marshal error: %v)", err)
	}
}

func TestParser_ComprehensiveSyntheticEvents(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	// prepare .git/config to validate gitRemoteUrl detection
	gitCfgDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitCfgDir, 0o755); err != nil {
		t.Fatalf("mkdir .git dir: %v", err)
	}
	remoteURL := "git@github.com:org/repo.git"
	cfgContent := "[remote \"origin\"]\n    url = " + remoteURL + "\n"
	if err := os.WriteFile(filepath.Join(gitCfgDir, "config"), []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("write .git/config: %v", err)
	}

	// Build synthetic records covering read/write/apply_diff and various field placements
	recs := []map[string]interface{}{
		{
			"type":      "assistant",
			"uuid":      "a1",
			"cwd":       tmpDir,
			"sessionId": "sess123",
			"timestamp": "2025-01-01T00:00:00Z",
			"message": map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{"type": "tool_use", "name": "Read"},
				},
			},
		},
		{
			"parentUuid": "a1",
			"timestamp":  "2025-01-01T00:00:01Z",
			"toolUseResult": map[string]interface{}{
				"type": "text",
				"file": map[string]interface{}{
					"filePath": "fileA.txt",
					"content":  "hello世界\n", // 8 runes (5+2+1)
					"numLines": float64(2),
				},
			},
		},
		{
			"type":      "assistant",
			"uuid":      "b1",
			"cwd":       tmpDir,
			"sessionId": "sess123",
			"timestamp": "2025-01-01T00:00:02Z",
			"message": map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{"type": "tool_use", "name": "Write"},
				},
			},
		},
		{
			"parentUuid": "b1",
			"timestamp":  "2025-01-01T00:00:03Z",
			"toolUseResult": map[string]interface{}{
				"type":     "create",
				"filePath": "fileB.txt",
				"content":  "line1\nline2\n", // 12 runes, 3 lines
			},
		},
		{
			"type":      "assistant",
			"uuid":      "c1",
			"cwd":       tmpDir,
			"sessionId": "sess123",
			"timestamp": "2025-01-01T00:00:04Z",
			"message": map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{"type": "tool_use", "name": "Edit"},
				},
			},
		},
		{
			"parentUuid": "c1",
			"timestamp":  "2025-01-01T00:00:05Z",
			"toolUseResult": map[string]interface{}{
				"filePath":  "fileB.txt",
				"oldString": "old content",
				"newString": "new content",
			},
		},
		{
			"type":      "assistant",
			"uuid":      "d1",
			"cwd":       tmpDir,
			"sessionId": "sess123",
			"timestamp": "2025-01-01T00:00:06Z", // 最后一个时间戳
			"message": map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "tool_use",
						"name": "Bash",
						"input": map[string]interface{}{
							"command":     "echo hello",
							"description": "test command",
						},
					},
				},
			},
		},
	}

	analysis := telemetry.AnalyzeConversations(recs)
	if len(analysis.Records) != 1 {
		t.Fatalf("expected 1 analysis record, got %d", len(analysis.Records))
	}
	record := analysis.Records[0]

	// Totals
	if record.TotalUniqueFiles != 2 {
		t.Errorf("TotalUniqueFiles expected 2, got %d", record.TotalUniqueFiles)
	}
	if record.TotalReadCharacters != 8 {
		t.Errorf("TotalReadCharacters expected 8, got %d", record.TotalReadCharacters)
	}
	if record.TotalWriteLines != 3 {
		t.Errorf("TotalWriteLines expected 3, got %d", record.TotalWriteLines)
	}
	if record.TotalWriteCharacters != 12 {
		t.Errorf("TotalWriteCharacters expected 12, got %d", record.TotalWriteCharacters)
	}
	if record.TotalDiffCharacters != 11 { // "new content" = 11 characters
		t.Errorf("TotalDiffCharacters expected 11, got %d", record.TotalDiffCharacters)
	}

	// Details
	if len(record.ReadFileDetails) != 1 || record.ReadFileDetails[0].FilePath != "fileA.txt" {
		t.Errorf("ReadFileDetails mismatch: %+v", record.ReadFileDetails)
	}
	if len(record.WriteToFileDetails) != 1 || record.WriteToFileDetails[0].FilePath != "fileB.txt" {
		t.Errorf("WriteToFileDetails mismatch: %+v", record.WriteToFileDetails)
	}
	if record.WriteToFileDetails[0].LineCount != 3 {
		t.Errorf("Write LineCount expected 3, got %d", record.WriteToFileDetails[0].LineCount)
	}
	if len(record.ApplyDiffDetails) != 1 || record.ApplyDiffDetails[0].FilePath != "fileB.txt" {
		t.Errorf("ApplyDiffDetails mismatch: %+v", record.ApplyDiffDetails)
	}
	if len(record.RunCommandDetails) != 1 || record.RunCommandDetails[0].Command != "echo hello" {
		t.Errorf("RunCommandDetails mismatch: %+v", record.RunCommandDetails)
	}

	// Tool call counts
	if record.ToolCallCounts.Read != 1 || record.ToolCallCounts.Write != 1 || record.ToolCallCounts.Edit != 1 || record.ToolCallCounts.Bash != 1 {
		t.Errorf("ToolCallCounts mismatch: %+v", record.ToolCallCounts)
	}

	// Context fields
	if record.TaskID != "sess123" {
		t.Errorf("taskId expected 'sess123', got %s", record.TaskID)
	}
	if record.FolderPath != tmpDir {
		t.Errorf("folderPath expected %s, got %s", tmpDir, record.FolderPath)
	}
	// last timestamp should be from 00:00:06Z (2025-01-01T00:00:06Z)
	wantLast, _ := time.Parse(time.RFC3339, "2025-01-01T00:00:06Z")
	if record.Timestamp != wantLast.Unix() {
		t.Errorf("timestamp mismatch, expected %d, got %d", wantLast.Unix(), record.Timestamp)
	}
	if record.GitRemoteURL != remoteURL {
		t.Errorf("gitRemoteUrl expected %s, got %s", remoteURL, record.GitRemoteURL)
	}
}

func TestParser_EmptyRecords_ReturnsEmpty(t *testing.T) {
	analysis := telemetry.AnalyzeConversations(nil)
	if len(analysis.Records) != 1 {
		t.Fatalf("expected 1 record (empty) for empty input, got %d", len(analysis.Records))
	}
	// Empty input should still create one record with zeros
	record := analysis.Records[0]
	if record.TotalUniqueFiles != 0 {
		t.Errorf("expected empty totals for empty input")
	}
}

// Integration tests that execute the binary and hit network are purposely omitted
// to keep tests hermetic. End-to-end behavior is covered by unit tests using
// telemetry.AnalyzeConversations and real sample JSONL lines.

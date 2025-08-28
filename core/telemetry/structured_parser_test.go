package telemetry

import (
	"testing"
)

func TestStructuredParser(t *testing.T) {
	// Create a sample JSONL record similar to the Python example
	sampleRecord := map[string]interface{}{
		"parentUuid":  nil,
		"isSidechain": false,
		"userType":    "user",
		"cwd":         "/workspace",
		"sessionId":   "test-session-123",
		"version":     "1.0.0",
		"gitBranch":   "main",
		"type":        "assistant",
		"uuid":        "msg-123",
		"timestamp":   "2025-01-01T12:00:00.000Z",
		"message": map[string]interface{}{
			"id":    "msg-123",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-3-sonnet",
			"content": []interface{}{
				map[string]interface{}{
					"type": "tool_use",
					"name": "Read",
					"id":   "tool-1",
					"input": map[string]interface{}{
						"file_path": "/workspace/test.txt",
					},
				},
			},
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage": map[string]interface{}{
				"input_tokens":                   10,
				"cache_creation_input_tokens":    0,
				"cache_read_input_tokens":        0,
				"output_tokens":                  5,
			},
		},
		"toolUseResult": map[string]interface{}{
			"type": "text",
			"file": map[string]interface{}{
				"filePath":   "/workspace/test.txt",
				"content":    "Hello, World!\nThis is a test file.",
				"numLines":   2,
				"startLine":  1,
				"totalLines": 2,
			},
		},
	}

	// Test the structured parser
	parser := NewStructuredParser()
	err := parser.ParseConversationLog(sampleRecord)
	if err != nil {
		t.Fatalf("Failed to parse conversation log: %v", err)
	}

	// Get the analysis record
	record := parser.GetAnalysisRecord()

	// Verify the results
	if record.TaskID != "test-session-123" {
		t.Errorf("Expected TaskID 'test-session-123', got '%s'", record.TaskID)
	}

	if record.FolderPath != "/workspace" {
		t.Errorf("Expected FolderPath '/workspace', got '%s'", record.FolderPath)
	}

	if record.ToolCallCounts.Read != 1 {
		t.Errorf("Expected Read tool count 1, got %d", record.ToolCallCounts.Read)
	}

	if len(record.ReadFileDetails) != 1 {
		t.Errorf("Expected 1 read detail, got %d", len(record.ReadFileDetails))
	}

	if record.ReadFileDetails[0].FilePath != "/workspace/test.txt" {
		t.Errorf("Expected file path '/workspace/test.txt', got '%s'", record.ReadFileDetails[0].FilePath)
	}

	if record.TotalReadCharacters != 34 {
		t.Errorf("Expected 34 characters, got %d", record.TotalReadCharacters)
	}
}

func TestAnalyzeConversations(t *testing.T) {
	// Test the main analysis function
	records := []map[string]interface{}{
		{
			"type":        "assistant",
			"uuid":        "msg-1",
			"sessionId":   "test-session",
			"cwd":         "/workspace",
			"timestamp":   "2025-01-01T12:00:00.000Z",
			"message": map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "tool_use",
						"name": "Write",
						"id":   "tool-1",
					},
				},
			},
			"toolUseResult": map[string]interface{}{
				"type":     "create",
				"filePath": "/workspace/new.txt",
				"content":  "New file content\nSecond line",
			},
		},
	}

	results := AnalyzeConversations(records)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	record := results[0]
	if record.ToolCallCounts.Write != 1 {
		t.Errorf("Expected Write tool count 1, got %d", record.ToolCallCounts.Write)
	}

	if len(record.WriteToFileDetails) != 1 {
		t.Errorf("Expected 1 write detail, got %d", len(record.WriteToFileDetails))
	}

	if record.WriteToFileDetails[0].Content != "New file content\nSecond line" {
		t.Errorf("Expected content 'New file content\\nSecond line', got '%s'", record.WriteToFileDetails[0].Content)
	}
}

func TestParseTimestamp(t *testing.T) {
	// Test timestamp parsing
	timestamp := "2025-01-01T12:00:00.000Z"
	expected := int64(1735732800000) // Expected Unix milliseconds

	result := ParseTimestamp(timestamp)
	if result != expected {
		t.Errorf("Expected timestamp %d, got %d", expected, result)
	}
}

func TestCountLines(t *testing.T) {
	// Test line counting
	testCases := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"single line", 1},
		{"line1\nline2", 2},
		{"line1\nline2\n", 3}, // trailing newline
		{"line1\nline2\nline3", 3},
	}

	for _, tc := range testCases {
		result := CountLines(tc.input)
		if result != tc.expected {
			t.Errorf("For input '%s', expected %d lines, got %d", tc.input, tc.expected, result)
		}
	}
}
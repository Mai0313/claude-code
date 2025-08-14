# AI Coding Agent Instructions for claude_analysis

## Project Overview
A command-line Go utility that reads JSON from stdin, enriches it with user metadata, and posts to a telemetry API. This is a Go port of a Python `post_hook.py` script, designed for cross-platform deployment with static binaries.

## Architecture & Core Components

### Single Binary Design
- **Main module**: `claude_analysis.go` - entire application logic in one file
- **Entry point**: `readStdinAndSave()` function handles the complete workflow
- **No external dependencies** - uses only Go standard library (`encoding/json`, `net/http`, `os/user`)

### Data Flow Pattern
```
STOP mode: stdin dict → parse transcript_path → read JSONL → aggregate → POST → return JSON
POST_TOOL mode: stdin JSON lines → aggregate directly (no file read) → POST → return JSON
```

Critical API details embedded in code:
- **Endpoint**: `https://gaia.mediatek.inc/o11y/upload_locs`
- **Headers**: `Content-Type: application/json` + `X-User-Id: <username>`
- **Timeout**: 10 seconds hardcoded

## Build System Conventions

### Makefile-Driven Workflow
- `make build` - standard single-platform build to `build/` directory
- `make build-all` - cross-compile for 6 platforms (Linux amd64/arm64, Windows amd64, macOS amd64/arm64)
- `make run` - build and execute (useful for testing with piped input)

### Platform Naming Convention
Binaries follow pattern: `claude_analysis-{os}-{arch}[.exe]`
- Current platform: `build/claude_analysis`
- Cross-compiled: `build/claude_analysis-linux-amd64`, etc.

## Error Handling Pattern
Uses explicit error wrapping with `fmt.Errorf()` and `%w` verb:
```go
return nil, fmt.Errorf("failed to parse JSON: %w", err)
```

All errors written to stderr, successful JSON output to stdout.

## Key Implementation Details

### User Context Injection
- Uses `os/user.Current()` to get system username
- Username becomes `X-User-Id` header value (not request body field)
- Cross-platform compatible user detection

### JSON Processing Approach
- Unmarshals to `map[string]interface{}` for flexibility
- Empty JSON input → empty response (early return)
- Pretty-prints response with 2-space indentation
- Reads JSONL transcript via `telemetry.ReadJSONL(path)` then aggregates with `telemetry.AggregateConversationStats(records)` (STOP mode)
- Alternatively aggregates directly from stdin JSON lines when `MODE=POST_TOOL`

#### Aggregation Output Schema
`records` is now an array containing one `ApiConversationStats` object with fields:
- `totalUniqueFiles`, `totalWriteLines`, `totalReadCharacters`, `totalWriteCharacters`, `totalDiffCharacters`
- `writeToFileDetails[]`, `readFileDetails[]`, `applyDiffDetails[]`
- `toolCallCounts`, `taskId`, `timestamp`, `folderPath`, `gitRemoteUrl`

### HTTP Client Configuration
- Custom client with 10-second timeout
- No retry logic or connection pooling
- Synchronous request/response pattern

## Development Workflows

### Testing Input/Output
```bash
echo '{"test": "data"}' | make run
cat sample.json | ./build/claude_analysis
# For JSONL aggregation (stdin may be Python-style dict)
echo "{'transcript_path':'/abs/path/tests/test_conversation.jsonl'}" | ./build/claude_analysis
# For POST_TOOL mode (stdin is JSON lines)
MODE=POST_TOOL ./build/claude_analysis <<'EOF'
{"type":"assistant","uuid":"u1","cwd":"/tmp/ws","sessionId":"s1","timestamp":"2025-01-01T00:00:00Z","message":{"content":[{"type":"tool_use","name":"Read"}]}}
{"parentUuid":"u1","timestamp":"2025-01-01T00:00:01Z","toolUseResult":{"filePath":"a.txt","content":"hello"}}
EOF
```

### Code Formatting
Always run `make fmt` before commits (uses `go fmt ./...`)

## Project-Specific Conventions

### File Organization
- Single-file application approach (no packages/modules)
- Build artifacts isolated in `build/` directory
- No separate config files - all settings hardcoded

### Variable Naming
- Uses `sessionDict` for input data (legacy from Python version)
- `responseDict` for API response
- Snake_case for JSON, camelCase for Go variables

## Integration Points

### API Contract
- Expects JSON input via stdin (Python 字典格式亦可；會自動轉換)，需包含 `transcript_path`
- API endpoint is environment-specific (hardcoded to `mtktma:8116`)
- No authentication beyond username header
- Response structure varies but always JSON
- Response payload sent to API has fields:
  - `user` from OS username, `records` from aggregated list, `extensionName` = `Claude-Code`, `machineId` from system, `insightsVersion` = `v0.0.1`

When modifying this codebase:
1. Maintain single-file simplicity - avoid splitting into packages
2. Keep API endpoint/timeout configurable only via code changes
3. Preserve cross-platform build capability in Makefile
4. Use explicit error handling with context wrapping
5. Test with actual JSON payloads via stdin, not unit tests
6. Add new parsing/aggregation under `core/telemetry/` (e.g. `parser.go`) and keep `main.go` minimal

# AI Coding Agent Instructions for claude_analysis

## Project Overview
A dual-purpose Go application that processes Claude Code conversation telemetry data. Functions as both a data aggregation CLI tool and an installer for Claude Code monitoring infrastructure. Replaces Python-based telemetry hooks with cross-platform Go binaries.

## Architecture & Core Components

### Modular Design Pattern
- **cmd/claude_analysis/**: Main telemetry processor with stdin→API workflow
- **cmd/installer/**: Interactive installer for Claude Code setup with Node.js validation
- **core/config/**: Configuration management with .env support and OS detection
- **core/telemetry/**: Data processing pipeline (input→parsing→aggregation→HTTP client)

### Dual Operating Modes
```
STOP mode (default): stdin dict → extract transcript_path → read JSONL → aggregate → POST
POST_TOOL mode: stdin JSONL stream → aggregate directly → POST
```

Mode switching via environment variables (`MODE=POST_TOOL`) or `.env` file in CWD.

### Critical Integration Points
- **API Endpoint**: `https://gaia.mediatek.inc/o11y/upload_locs` (hardcoded in config)
- **Headers**: Automatic `X-User-Id` injection from OS username via `os/user.Current()`
- **External Dependency**: `github.com/denisbrodbeck/machineid` for device fingerprinting

## Build System Conventions

### Makefile-Driven Multi-Target Workflow
- `make build` - builds both `claude_analysis` and `installer` binaries for local platform
- `make build-all` - cross-compiles for 5 platforms (Linux amd64/arm64, Windows amd64, macOS amd64/arm64)  
- `make package-all` - creates platform-specific ZIP packages with READMEs
- `make test` / `make test-verbose` - runs Go tests with coverage

### Release Packaging Convention
Output ZIP pattern: `Claude-Code-Installer-{platform}.zip` containing:
- `claude_analysis[.exe]` - telemetry processor
- `installer[.exe]` - interactive setup tool  
- `README*.md` files for documentation

### Cross-Platform Binary Naming
- Local build: `build/claude_analysis`, `build/installer`
- Cross-compiled: `build/claude_analysis-{os}-{arch}[.exe]`

## Data Processing Pipeline

### JSONL Conversation Parsing
Core function: `telemetry.AggregateConversationStats()` processes Claude Code conversation logs
- **Input**: Array of event maps from conversation JSONL
- **Output**: Single `ApiConversationStats` object with aggregated metrics
- **Key patterns**: Tracks tool usage (`Read`, `Write`, `ApplyDiff`), file operations, character counts

### Python Dict Compatibility Layer
`telemetry.ExtractTranscriptPath()` handles Python-style dict input:
```python
{'transcript_path': '/path/to/conversation.jsonl'}  # Auto-converted to JSON
```

### Tool Call Aggregation Schema
Final payload structure sent to API:
```go
{
  "user": "<os-username>",
  "records": [ApiConversationStats], 
  "extensionName": "Claude-Code",
  "machineId": "<device-fingerprint>",
  "insightsVersion": "v0.0.1"
}
```

## Development Workflows

### Testing Input Modes
```bash
# STOP mode: Python dict with transcript path
echo "{'transcript_path':'/proj/ds906659/gai/claude_analysis/tests/test_conversation.jsonl'}" | make run

# POST_TOOL mode: Direct JSONL processing  
MODE=POST_TOOL make run <<EOF
{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read"}]}}
{"toolUseResult":{"filePath":"test.txt","content":"hello world"}}
EOF

# Cross-platform testing
make build-all && ./build/claude_analysis-linux-amd64 < input.json
```

### Configuration Management
Mode selection priority: `MODE` env var → `mode` env var → `.env` file → default "STOP"
- **Custom .env parsing**: No external dependencies, manual key=value parsing with quote trimming
- **Runtime config injection**: Username, machine ID, API endpoint baked into `config.Default()`

### Error Handling Pattern  
Consistent error wrapping with context:
```go
return nil, fmt.Errorf("failed to parse JSON: %w", err)
```
All errors to stderr, successful JSON to stdout with pretty-printing.

## Project-Specific Conventions

### Single-Module Architecture
- All business logic in `main.go` entry points
- Shared functionality extracted to `core/` packages
- No third-party dependencies except `github.com/denisbrodbeck/machineid`

### Git Integration
`telemetry.getGitRemoteOriginURL()` parses `.git/config` manually to extract `remote.origin.url`
- Used for workspace context in telemetry payload
- Fallback-safe: returns empty string on any parsing failure

### Installer Integration  
`cmd/installer/` provides interactive Claude Code setup:
- Node.js detection and installation guidance
- Claude Code CLI installation via npm
- Settings.json hook configuration for telemetry collection

### Multi-Language Support
README files in English, 简体中文, and 繁體中文 - maintain consistency across all three when making documentation changes.

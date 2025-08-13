# Claude Analysis - Go Version

This is the Go language port of the original Python `post_hook.py` script, now renamed to `claude_analysis`.

## Features

- Reads JSON data from standard input
- Gets current user information  
- Sends data to HTTP API endpoint
- Returns API response
- Cross-platform compilation support

## Project Structure

This project follows the standard Go project layout:

```
claude_analysis/
├── cmd/claude_analysis/        # Main application entry point
├── core/                   # Private application code
│   ├── config/                # Configuration management
│   └── telemetry/             # Telemetry functionality
├── pkg/                       # Public library code
├── build/                     # Build outputs
├── docs/                      # Documentation
├── scripts/                   # Build and utility scripts
└── ...
```

See [docs/project_structure.md](docs/project_structure.md) for detailed structure explanation.

## Build Instructions

### Prerequisites
- Go 1.21 or later installed
- Make (optional, for using Makefile)

### Build for current platform
```bash
# Using Makefile (recommended)
make build

# Or using the build script (includes version info)
./scripts/build.sh

# Or directly with go
mkdir -p build && go build -o build/claude_analysis ./cmd/claude_analysis
```

### Build for multiple platforms
```bash
# Build for all supported platforms
make build-all
```

All binaries will be created in the `build/` directory:
- `build/claude_analysis` - Current platform
- `build/claude_analysis-linux-amd64` - Linux AMD64
- `build/claude_analysis-linux-arm64` - Linux ARM64  
- `build/claude_analysis-windows-amd64.exe` - Windows AMD64
- `build/claude_analysis-darwin-amd64` - macOS Intel
- `build/claude_analysis-darwin-arm64` - macOS Apple Silicon

## Usage

The Go version works exactly like the Python version:

```bash
# Pipe JSON data to the program
echo '{"key": "value"}' | ./build/claude_analysis

# Or from a file
cat data.json | ./build/claude_analysis

# Using make run (builds and runs)
make run
```

## Key Differences from Python Version

1. **Error Handling**: More explicit error handling and reporting
2. **Typing**: Strong static typing instead of Python's dynamic typing
3. **Performance**: Compiled binary with better performance
4. **Cross-platform**: Easy compilation for multiple platforms
5. **Dependencies**: No external dependencies (uses only Go standard library)

## API Details

- **Endpoint**: `http://mtktma:8116/tma/sdk/api/logs`
- **Method**: POST
- **Headers**: 
  - `Content-Type: application/json`
  - `X-User-Id: <current_username>`
- **Timeout**: 10 seconds

## Development

### Format code
```bash
make fmt
```

### Run tests
```bash
# Run integration tests with sample data
./scripts/test.sh

# Run unit tests (future)
make test
```

### Clean build artifacts  
```bash
make clean
```

### Install to system (optional)
```bash
make install
```

## Platform-specific Notes

- **Linux**: Uses standard user lookup
- **Windows**: Compatible with Windows user system
- **macOS**: Works on both Intel and Apple Silicon Macs

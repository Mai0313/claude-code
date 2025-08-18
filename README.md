# Claude Code CLI Usage Guide

English | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md)

## Introduction

Claude Code is Anthropic's official CLI tool that provides AI programming assistance and interactive development support. This installer provides automated setup with intelligent network detection and optional JWT authentication.

---

## Authentication Options

### Option 1: JWT Token Authentication (Recommended)
The installer can automatically obtain and configure JWT tokens for seamless authentication:

1. Run the installer
2. When prompted, choose "y" for JWT token configuration
3. Enter your MediaTek credentials
4. The installer will automatically:
   - Detect the best available MLOP endpoint
   - Obtain a JWT token
   - Configure authentication headers

### Option 2: Manual API Key Setup
If you prefer manual configuration or encounter issues with JWT authentication:

1. Visit [MediaTek MLOP Gateway for OA](https://mlop-azure-gateway.mediatek.inc/auth/login) / [MediaTek MLOP Gateway for SWRD](https://mlop-azure-rddmz.mediatek.inc/auth/login) to login
2. Obtain your GAISF API key
3. Manually configure the key in your settings

**Note**: The installer automatically detects network connectivity and chooses between HTTP/HTTPS protocols for optimal compatibility.

## Installation Features

The installer includes advanced capabilities for reliable setup:

### Smart Dependency Management
- **Node.js 22+ Detection**: Automatically checks and installs the required Node.js version
- **Platform-Specific Installation**: 
  - **macOS**: Uses Homebrew for automatic installation
  - **Linux**: Supports multiple package managers (apt, dnf, yum, pacman)
  - **Windows**: Provides direct download links with guided installation

### Intelligent Network Detection
- **Multi-Registry Support**: Automatically tests MediaTek internal npm registries for optimal download speed
- **Endpoint Auto-Selection**: Detects the best available MLOP gateway endpoint
- **Fallback Mechanisms**: Seamlessly switches to backup servers if primary connections fail

### Configuration Management
- **System-Level Installation**: Supports both user-level and system-wide configurations
- **Multi-Platform Binaries**: Installs platform-specific claude_analysis binaries with proper naming
- **Managed Settings**: Automatically generates optimized settings.json with telemetry and MCP server support

---

## Installation

### Step 1: Download
Download the latest installer package from:
https://gitea.mediatek.inc/IT-GAIA/claude-code-monitor/releases

### Step 2: Extract and Run
1. Extract the downloaded zip file
2. Open a terminal/command prompt in the extracted folder
3. Run the installer:
   - **Linux/macOS**: `./installer`
   - **Windows**: `installer.exe`

### Step 3: Configure Authentication
During installation, you'll be prompted to configure authentication:

1. **JWT Token Setup (Recommended)**:
   - Choose "y" when asked about JWT token configuration
   - Enter your MediaTek username and password
   - The installer will securely obtain and configure your JWT token

2. **Skip JWT Setup**:
   - Choose "N" to skip JWT configuration
   - You can manually configure API keys later if needed

The installer automatically handles all technical configuration including:
- Network endpoint detection and selection
- Registry fallback configuration
- Platform-specific binary installation
- Optimized Claude Code settings

## Important for Windows Users

**⚠️ Windows users must install Node.js manually first!**

If you don't have Node.js installed, the installer will:
1. Show you the direct download link for Node.js
2. Exit so you can install it
3. Ask you to run the installer again after Node.js is installed

**Quick steps for Windows:**
1. Download Node.js from the link provided by the installer
2. Install the Node.js MSI package
3. Restart your command prompt
4. Run the installer again

## System Requirements

- **Operating System**: Windows, macOS, or Linux
- **Node.js**: Version 22 or higher (automatically installed on macOS/Linux)
- **Internet connection**: Required for downloading components and authentication
- **Credentials**: MediaTek account for JWT authentication (optional but recommended)

## Installation Locations

The installer creates the following files and directories:

### User-Level Installation (Default)
- **Claude CLI**: Installed globally via npm
- **Configuration**: `~/.claude/settings.json`
- **Binary**: `~/.claude/claude_analysis-{platform}-{arch}[.exe]`

### System-Level Installation (When Available)
- **macOS**: `/Library/Application Support/ClaudeCode/`
- **Linux**: `/etc/claude-code/`
- **Windows**: `C:\ProgramData\ClaudeCode\`

The installer automatically chooses the best installation method based on system permissions and requirements.

## Troubleshooting

**Problem**: `claude --version` doesn't work after installation
**Solution**: Restart your terminal or add npm's global bin directory to your PATH

**Problem**: JWT authentication fails during installation
**Solution**: 
- Verify your MediaTek credentials are correct
- Check network connectivity to MLOP gateways
- Skip JWT setup and configure manually if needed

**Problem**: Node.js installation fails on Linux/macOS
**Solution**: 
- The installer tries multiple package managers automatically
- If all fail, manually install Node.js 22+ from https://nodejs.org/
- Re-run the installer after manual Node.js installation

**Problem**: Installation fails with network errors
**Solution**: 
- The installer automatically tries backup registries and endpoints
- Ensure you have internet connectivity
- Try running behind a corporate firewall with appropriate proxy settings

**Problem**: Need to reinstall or update
**Solution**: You can safely run the installer multiple times - it detects existing installations and updates appropriately

**Problem**: Configuration file not being read
**Solution**: The installer creates configuration in multiple locations. Check:
- `~/.claude/settings.json` (user-level)
- System-level managed settings (OS-specific paths)
- Refer to [official configuration documentation](https://docs.anthropic.com/en/docs/claude-code/settings#configuration-file) for additional locations

## Additional Resources

- [Official Documentation](https://docs.anthropic.com/en/docs/claude-code)
- [Settings Documentation](https://docs.anthropic.com/en/docs/claude-code/settings)
- [Sub-agents Feature](https://docs.anthropic.com/en/docs/claude-code/sub-agents) - Explore agent capabilities for specialized tasks
- [MCP Integration](https://docs.anthropic.com/en/docs/claude-code/mcp) - Learn about Model Context Protocol support

## Getting Help

If you encounter issues:
1. Make sure you have internet connectivity
2. Try running the installer as administrator (Windows) or with `sudo` (macOS/Linux) for system-wide installation
3. Check that Node.js is properly installed with `node --version` (should be 22+)
4. For JWT authentication issues, verify your MediaTek account credentials
5. Check the installer output for specific error messages and retry with different options if needed

## New Features in This Version

### Enhanced Authentication
- **Automatic JWT Token Management**: Secure credential handling with automatic token refresh
- **Smart Endpoint Detection**: Automatically selects the best available MLOP gateway
- **Credential Security**: Password input is hidden on Unix systems for security

### Improved Network Reliability
- **Multi-Registry Support**: Automatic fallback between MediaTek internal npm registries
- **Connectivity Testing**: Pre-installation network checks ensure optimal download paths
- **Protocol Auto-Selection**: Intelligent HTTP/HTTPS selection based on network conditions

### Advanced Installation Options
- **Platform-Specific Binaries**: Properly named binaries for each platform and architecture
- **System-Level Configuration**: Support for enterprise-wide managed settings
- **Dependency Auto-Installation**: Automated Node.js setup across all supported platforms

### Configuration Enhancements
- **Optimized Default Settings**: Pre-configured telemetry, MCP servers, and co-authoring features
- **Flexible Configuration Paths**: Support for both user and system-level configuration files
- **Interactive Setup**: User-friendly prompts for authentication and configuration choices

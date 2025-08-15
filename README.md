# Claude Code CLI Usage Guide

English | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md)

## Introduction

Claude Code is Anthropic's official CLI tool that provides AI programming assistance and interactive development support.

---

## Getting API Key

**Important**: Before using Claude Code, please complete the following steps:

1. Visit [MediaTek MLOP Gateway for OA](https://mlop-azure-gateway.mediatek.inc/auth/login) / [MediaTek MLOP Gateway for SWRD](https://mlop-azure-rddmz.mediatek.inc/auth/login) to login
2. Obtain your GAISF API key
3. Keep the key secure for future use

**Note**: Due to SSL certificate configuration issues, the URLs in this documentation use HTTP instead of HTTPS to ensure compatibility across different network environments.

---

**Configuration Instructions:**
- Replace `<<Your GAISF API Key>>` with your actual API key
- `ANTHROPIC_BEDROCK_BASE_URL` is set to use HTTP instead of HTTPS due to SSL certificate configuration requirements
- **Important Reminder**: In some cases, Claude Code may not be able to read the `~/.claude/settings.json` configuration file. If you encounter configuration issues, please refer to the [official configuration file documentation](https://docs.anthropic.com/en/docs/claude-code/settings#configuration-file) for other configuration file locations and troubleshooting steps

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

### Step 3: Follow the prompts
The installer will guide you through the setup process and display progress messages.

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
- **Node.js**: LTS version (will be installed automatically on macOS/Linux)
- **Internet connection**: Required for downloading components

## What gets installed where?

After successful installation, you'll find:
- Claude Code CLI available globally (try `claude --version`)
- Configuration files in your home directory under `.claude/`

## Troubleshooting

**Problem**: `claude --version` doesn't work after installation
**Solution**: Restart your terminal or add npm's global bin directory to your PATH

**Problem**: Installation fails on the first try
**Solution**: The installer automatically retries with backup servers - just wait for it to complete

**Problem**: Need to reinstall or update
**Solution**: You can safely run the installer multiple times - it won't break anything

## Additional Resources

- [Official Documentation](https://docs.anthropic.com/en/docs/claude-code)
- [Settings Documentation](https://docs.anthropic.com/en/docs/claude-code/settings)
- [Sub-agents Feature](https://docs.anthropic.com/en/docs/claude-code/sub-agents) - Explore agent capabilities for specialized tasks
- [MCP Integration](https://docs.anthropic.com/en/docs/claude-code/mcp) - Learn about Model Context Protocol support

## Getting Help

If you encounter issues:
1. Make sure you have internet connectivity
2. Try running the installer as administrator (Windows) or with `sudo` (macOS/Linux) for system-wide installation
3. Check that Node.js is properly installed with `node --version`

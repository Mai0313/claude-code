# Claude Code Installer

A simple installer that helps you set up Claude Code with telemetry analytics for your development workflow.

## What does this installer do?

This installer will automatically:
1. **Install Node.js** (if not present) - Required for Claude Code to run
2. **Install Claude Code CLI** - The main application
3. **Set up analytics integration** - Tracks your coding activity for insights
4. **Configure everything automatically** - No manual configuration needed

## How to use

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
- Analytics component ready to track your development activity

## Troubleshooting

**Problem**: `claude --version` doesn't work after installation
**Solution**: Restart your terminal or add npm's global bin directory to your PATH

**Problem**: Installation fails on the first try
**Solution**: The installer automatically retries with backup servers - just wait for it to complete

**Problem**: Need to reinstall or update
**Solution**: You can safely run the installer multiple times - it won't break anything

## Getting Help

If you encounter issues:
1. Make sure you have internet connectivity
2. Try running the installer as administrator (Windows) or with `sudo` (macOS/Linux) for system-wide installation
3. Check that Node.js is properly installed with `node --version`

# Install Claude Code with the Installer

English | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md)

## What this installer does

This guide shows how to install the Claude Code CLI using the bundled installer. The installer will:

1) Check Node.js (needs v22+). On macOS/Linux it tries to install it automatically; on Windows it shows a download link and exits so you can install Node.js, then rerun.
2) Install the Claude Code CLI globally via npm: @anthropic-ai/claude-code.
3) Create ~/.claude/settings.json with sensible defaults and optional authentication.

The installer also:

- Auto-detects internal registries and MLOP gateways to improve reliability
- If Claude Code is already installed, runs an update to get the latest version (equivalent to `claude update`)

---

## Step-by-step installation

### 1) Download
Get the latest installer for your OS from:
https://gitea.mediatek.inc/IT-GAIA/claude-code/releases

Choose the zip that matches your platform (Windows, macOS Intel/Apple Silicon, or Linux x64/ARM64).

### 2) Extract
Unzip the downloaded file to a folder you can access from a terminal/command prompt.

### 3) Run the installer
- macOS/Linux
   - Open Terminal in the unzipped folder
   - If needed, make it executable: chmod +x ./installer
   - Run: ./installer
   - macOS: you can also double-click the "installer" to launch it in Terminal

- Windows
   - Double-click installer.exe, or run it from PowerShell

### 4) Follow the prompts
- If Node.js is missing or below v22:
   - macOS/Linux: the installer will attempt to install it (may ask for your sudo password or use Homebrew/apt/dnf/etc.)
   - Windows: you’ll see a download link; install Node.js from that link, then run the installer again

- Authentication setup (recommended):
   - When asked “Do you want to configure GAISF token for API authentication? (y/N)”, choose y
   - Enter your MediaTek username and password
   - If automatic token retrieval fails, you’ll be asked to paste an API key you can get after logging in to:
      - OA: https://mlop-azure-gateway.mediatek.inc/auth/login
      - SWRD: https://mlop-azure-rddmz.mediatek.inc/auth/login
   - Choose N to skip for now; you can add credentials later in ~/.claude/settings.json

### 5) Verify
- Open a new terminal and run: claude --version
- You should see a version printed. If not, see Troubleshooting below.

---

## Requirements
- Windows, macOS, or Linux
- Node.js v22 or newer (macOS/Linux can be installed by the installer; Windows must install manually if missing)
- Internet access (for downloads and authentication)

## Troubleshooting

- “claude: command not found”
   - Restart your terminal so PATH updates take effect
   - Ensure npm’s global bin is on PATH

- Node.js installation troubles (macOS/Linux)
   - Manually install Node.js v22+ from https://nodejs.org/ and rerun the installer

- Node.js on Windows
   - Use the link shown by the installer to download and install Node.js, then rerun

- Authentication issues
   - Verify your MediaTek credentials
   - If GAISF token setup fails, use the manual API key fallback as prompted

## Links
- Official Claude Code docs: https://docs.anthropic.com/en/docs/claude-code
- Settings: https://docs.anthropic.com/en/docs/claude-code/settings

---

## Expected files after installation

Directory layout:

```
├── .claude
│   ├── claude_analysis-linux-amd64
│   └── settings.json
```

Sample `~/.claude/settings.json`:

```json
{
   "env": {
      "ANTHROPIC_BEDROCK_BASE_URL": "https://mlop-azure-gateway.mediatek.inc",
      "ANTHROPIC_CUSTOM_HEADERS": "api-key: <<api_key>>",
      "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
      "CLAUDE_CODE_ENABLE_TELEMETRY": "1",
      "CLAUDE_CODE_SKIP_BEDROCK_AUTH": "1",
      "CLAUDE_CODE_USE_BEDROCK": "1",
      "DISABLE_TELEMETRY": "1"
   },
   "includeCoAuthoredBy": true,
   "enableAllProjectMcpServers": true,
   "hooks": {
      "Stop": [
         {
            "matcher": "*",
            "hooks": [
               {
                  "type": "command",
                  "command": "/proj/ds906659/.claude/claude_analysis-linux-amd64"
               }
            ]
         }
      ]
   }
}
```

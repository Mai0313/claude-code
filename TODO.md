幫我透過 Golang 做一個腳本 用途是自動安裝 Claude Code CLI 工具
只需要寫在一個檔案就可以了 因為他只是一個簡單腳本
首先 檢查使用者的作業系統是 Windows, macOS 還是 Linux 做以下對應步驟

1. 安裝對應作業系統的 `node.js` (v22.18.0 LTS 版本)
2. 透過 `node -v` 和 `npm -v` 確認是否安裝成功
3. 安裝 `@anthropic-ai/claude-code` 套件
  - `npm install -g @anthropic-ai/claude-code --registry=http://oa-mirror.mediatek.inc/repository/npm`
  - 透過 `claude --version` 來確認是否安裝成功
5. 將 `claude_analysis` 移動到 `~/.claude` 資料夾內, 有下列幾種版本可以下載 需要額外判斷系統
  - https://gitea.mediatek.inc/IT-GAIA/claude-code-monitor/releases/download/v0.0.2/claude_analysis-darwin-amd64
  - https://gitea.mediatek.inc/IT-GAIA/claude-code-monitor/releases/download/v0.0.2/claude_analysis-darwin-arm64
  - https://gitea.mediatek.inc/IT-GAIA/claude-code-monitor/releases/download/v0.0.2/claude_analysis-linux-amd64
  - https://gitea.mediatek.inc/IT-GAIA/claude-code-monitor/releases/download/v0.0.2/claude_analysis-linux-arm64
  - https://gitea.mediatek.inc/IT-GAIA/claude-code-monitor/releases/download/v0.0.2/claude_analysis-windows-amd64.exe
4. 安裝成功以後將這個config寫入到 `~/.claude/settings.json`
  - Example:
  ```json
  {
    "env": {
      "DISABLE_TELEMETRY": "1",
      "CLAUDE_CODE_USE_BEDROCK": "1",
      "ANTHROPIC_BEDROCK_BASE_URL": "http://mlop-azure-rddmz.mediatek.inc",
      "CLAUDE_CODE_ENABLE_TELEMETRY": "1",
      "CLAUDE_CODE_SKIP_BEDROCK_AUTH": "1",
      "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1"
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
              "command": "~/.claude/claude_analysis-linux-amd64"
            }
          ]
        }
      ]
    }
  }
  ```
- 注意 `"command": "~/.claude/claude_analysis-linux-amd64"` 這個路徑需要根據使用者的作業系統來調整
5. 最後確認 `~/.claude/settings.json` 是否寫入成功

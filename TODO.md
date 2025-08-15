幫我透過 Golang 做一個類似於入口腳本的 用途是自動安裝 Claude Code CLI 工具
只需要寫在一個檔案就可以了 因為他只是一個簡單腳本

修改 `claude_analysis` 的建置流程 我希望建置出來的東西可以放在 `build` 裡面 命名方式是 `Claude-Code-Installer-{platform version}.zip`
  - `Claude-Code-Installer-{platform version}.zip` 裡面包含以下幾個檔案
    - `claude_analysis` (這是主要的 CLI 工具 並且已編譯成可執行檔)
    - `settings.json` (這是配置檔)
    - `installer` (這是此次透過 Golang 寫的安裝腳本 並且已編譯成可執行檔)
      - 首先 檢查使用者的作業系統是 Windows, macOS 還是 Linux 安裝對應作業系統的 `node.js` (LTS 版本)
        - 先確認是否已經安裝過 `node.js` 如果有 就跳過安裝
        - 如果是 Windows 同時沒有安裝 `node.js`, 直接提示用戶去 `https://nodejs.org/dist/v22.18.0/node-v22.18.0-arm64.msi` 下載安裝 安裝好再重新執行腳本 腳本這時候就會直接從第二步開始
      - 安裝 `@anthropic-ai/claude-code` 套件
        - `npm install -g @anthropic-ai/claude-code --registry=...`
        - `registry` 先嘗試不使用任何 `registry`, 失敗後嘗試用 `http://oa-mirror.mediatek.inc/repository/npm`, 還是失敗就用 `http://swrd-mirror.mediatek.inc/repository/npm`
        - 透過 `claude --version` 來確認是否安裝成功
      - 將 `claude_analysis` 移動到 `~/.claude` 資料夾內
      - 將以下配置寫入到 `~/.claude/settings.json`

`settings.json` 範例:

- ANTHROPIC_BEDROCK_BASE_URL 會有兩種可能 可測試連通性來決定
  - https://mlop-azure-gateway.mediatek.inc
  - https://mlop-azure-rddmz.mediatek.inc

假設最一開始 `claude-code` 是透過 http://oa-mirror.mediatek.inc/repository/npm 安裝, `ANTHROPIC_BEDROCK_BASE_URL` 就要用 https://mlop-azure-gateway.mediatek.inc
反之, `claude-code` 如果是透過 http://swrd-mirror.mediatek.inc/repository/npm 安裝, `ANTHROPIC_BEDROCK_BASE_URL` 就要用 https://mlop-azure-rddmz.mediatek.inc

- `~/.claude/claude_analysis-linux-amd64` 取決於作業系統

> 所以 `settings.json` 可能要參考下方的範例來動態生成

```json
{
  "env": {
    "DISABLE_TELEMETRY": "1",
    "CLAUDE_CODE_USE_BEDROCK": "1",
    "ANTHROPIC_BEDROCK_BASE_URL": "http://mlop-azure-{XXX}.mediatek.inc",
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

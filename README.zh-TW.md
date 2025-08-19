# 使用安裝程式安裝 Claude Code

[English](README.md) | 繁體中文 | [简体中文](README.zh-CN.md)

## 安裝程式會做什麼

本指南說明如何用隨附的安裝程式安裝 Claude Code CLI。安裝程式會：

1) 檢查 Node.js（需要 v22+）。在 macOS/Linux 嘗試自動安裝；Windows 會顯示官方下載頁（https://nodejs.org/en/download/）並結束，請先安裝後再重跑。
2) 透過 npm 全域安裝 Claude Code CLI：@anthropic-ai/claude-code。
3) 建立 ~/.claude/settings.json，內含預設與可選擇的認證設定。

此外會自動偵測內部 npm registry 與 MLOP gateway，提升安裝成功率。

若系統已安裝 Claude Code，安裝程式會嘗試幫你更新到最新版（等同於執行 `claude update`）。

- 建立設定檔時，若 `~/.claude/settings.json` 已存在，安裝程式會先詢問是否覆蓋；同意後會先備份舊檔為 `settings.backup_YYYYMMDD_HHMMSS.json`，再寫入新檔。

---

## 逐步安裝

### 1) 下載
到以下頁面下載對應作業系統的安裝包：
https://gitea.mediatek.inc/IT-GAIA/claude-code/releases

請選擇符合您平台的 zip（Windows、macOS Intel/Apple Silicon、或 Linux x64/ARM64）。

### 2) 解壓縮
把 zip 解壓到方便從終端機/命令提示字元開啟的資料夾。

### 3) 執行安裝程式
- macOS/Linux
   - 在解壓資料夾開啟 Terminal
   - 如需，先給執行權限：chmod +x ./installer
   - 執行：./installer
   - macOS：也可直接雙擊「installer」在終端機啟動

- Windows
   - 直接雙擊 installer.exe，或從 PowerShell 執行

### 4) 依照提示操作
- 若未安裝或版本低於 v22 的 Node.js：
   - macOS/Linux：安裝程式會嘗試自動安裝（可能需要 sudo 密碼或使用 Homebrew/apt/dnf 等）
   - Windows：會顯示官方 Node.js 下載頁連結（https://nodejs.org/en/download/）；請先安裝 Node.js 後再重新執行安裝程式

- 認證設定（建議）：
   - 出現「Do you want to configure GAISF token for API authentication? (y/N)」時選 y
   - 終端機流程：輸入 MediaTek 帳號與密碼；若自動取得失敗，仍會請你貼上 GAISF token
   - 手動流程：開啟 GAISF 登入網址並登入，接著把 GAISF token 貼回安裝程式
   - GAISF 登入網址：
      - OA：https://mlop-azure-gateway.mediatek.inc/auth/login
      - SWRD：https://mlop-azure-rddmz.mediatek.inc/auth/login
   - 若暫不設定，選 N；之後可在 `~/.claude/settings.json` 補上 token

### 5) 驗證
- 開新終端機執行：claude --version
- 看到版本號即完成。若失敗，請見下方疑難排解。

---

## 需求
- Windows、macOS、或 Linux
- Node.js v22 或以上（macOS/Linux 可由安裝程式代裝；Windows 缺少時需自行安裝）
- 可連網（下載與認證）

## 疑難排解

- 出現「claude: command not found」
   - 重新開啟終端機讓 PATH 生效
   - 確認 npm 的 global bin 已加入 PATH

- macOS/Linux 安裝 Node.js 困難
   - 安裝程式在 Debian/Ubuntu 可能會自動嘗試 NodeSource 22.x。若仍失敗，請從 https://nodejs.org/ 手動安裝 v22+，再重跑安裝程式

- Windows 安裝 Node.js
   - 使用官方 Node.js 下載頁：https://nodejs.org/en/download/ 下載並安裝，完成後重跑

- 認證問題
   - 檢查 MediaTek 憑證
   - 若 GAISF token 設定失敗，請手動開啟 GAISF 登入網址並於提示時貼上 GAISF token

## 參考
- Claude Code 官方文件：https://docs.anthropic.com/zh-TW/docs/claude-code
- 設定說明：https://docs.anthropic.com/zh-TW/docs/claude-code/settings

---

## 安裝完成後的預期檔案

目錄結構：

```
├── .claude
│   ├── claude_analysis-linux-amd64
│   └── settings.json
```

範例 `~/.claude/settings.json`：

```json
{
   "env": {
      "ANTHROPIC_BEDROCK_BASE_URL": "https://mlop-azure-gateway.mediatek.inc",
   "ANTHROPIC_CUSTOM_HEADERS": "api-key: <<gaisf_token>>",
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

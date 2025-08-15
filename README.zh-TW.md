# Claude Code CLI 使用說明

[English](README.md) | 繁體中文 | [简体中文](README.zh-CN.md)

## 簡介

Claude Code 是 Anthropic 官方的 CLI 工具，提供 AI 編程助手和互動式開發支援。

---

## 取得 API 金鑰

**重要**：使用 Claude Code 前，請先完成以下步驟：

1. 前往 [MediaTek MLOP Gateway for OA](https://mlop-azure-gateway.mediatek.inc/auth/login) / [MediaTek MLOP Gateway for SWRD](https://mlop-azure-rddmz.mediatek.inc/auth/login)登入
2. 取得您的 GAISF API 金鑰
3. 妥善保存金鑰以供後續使用

**注意**：由於 SSL 憑證設定問題，本說明文件中的網址使用 HTTP 而非 HTTPS，以確保在不同網路環境下的相容性。

---

**設定說明：**
- 請將 `<<您的 GAISF API 金鑰>>` 替換為您的實際 API 金鑰
- `ANTHROPIC_BEDROCK_BASE_URL` 設定為使用 HTTP 而非 HTTPS，這是由於 SSL 憑證設定需求
- **重要提醒**：在某些情況下，Claude Code 可能無法讀取 `~/.claude/settings.json` 設定檔。如果您遇到設定問題，請參考 [官方設定檔說明文件](https://docs.anthropic.com/zh-TW/docs/claude-code/settings#%E8%A8%AD%E5%AE%9A%E6%AA%94%E6%A1%88) 了解其他設定檔位置和故障排除步驟

## 安裝

### 第一步：下載
從以下頁面下載最新的安裝套件：
https://gitea.mediatek.inc/IT-GAIA/claude-code-monitor/releases

### 第二步：解壓縮並執行
1. 解壓縮下載的 zip 檔案
2. 在解壓縮後的資料夾中開啟終端機/命令提示字元
3. 執行安裝程式：
   - **Linux/macOS**：`./installer`
   - **Windows**：`installer.exe`

### 第三步：依照提示操作
安裝程式會引導您完成設定過程並顯示進度訊息。

## Windows 使用者重要提醒

**⚠️ Windows 使用者必須先手動安裝 Node.js！**

如果您沒有安裝 Node.js，安裝程式會：
1. 向您顯示 Node.js 的直接下載連結
2. 結束程式，讓您先安裝 Node.js
3. 要求您在安裝 Node.js 後重新執行安裝程式

**Windows 快速步驟：**
1. 從安裝程式提供的連結下載 Node.js
2. 安裝 Node.js MSI 安裝套件
3. 重新啟動命令提示字元
4. 再次執行安裝程式

## 系統需求

- **作業系統**：Windows、macOS 或 Linux
- **Node.js**：LTS 版本（在 macOS/Linux 上會自動安裝）
- **網路連線**：下載元件時需要

## 安裝檔案位置

成功安裝後，您會發現：
- Claude Code CLI 全域可用（嘗試 `claude --version`）
- 設定檔位於您的主目錄下的 `.claude/` 資料夾

## 疑難排解

**問題**：安裝後 `claude --version` 無法運作
**解決方案**：重新啟動終端機或將 npm 的全域 bin 目錄加入 PATH

**問題**：第一次安裝失敗
**解決方案**：安裝程式會自動使用備用伺服器重試 - 請等待完成

**問題**：需要重新安裝或更新
**解決方案**：您可以安全地多次執行安裝程式 - 不會損壞任何東西

## 額外資源

- [官方文件](https://docs.anthropic.com/zh-TW/docs/claude-code)
- [設定文件](https://docs.anthropic.com/zh-TW/docs/claude-code/settings)
- [子代理功能](https://docs.anthropic.com/zh-TW/docs/claude-code/sub-agents) - 探索專業任務的代理功能
- [MCP 整合](https://docs.anthropic.com/zh-TW/docs/claude-code/mcp) - 了解模型上下文協定支援

## 取得協助

如果遇到問題：
1. 確保有網路連線
2. 嘗試以系統管理員身分執行安裝程式（Windows）或使用 `sudo`（macOS/Linux）進行系統層級安裝
3. 使用 `node --version` 檢查 Node.js 是否正確安裝

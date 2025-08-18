# Claude Code CLI 使用說明

[English](README.md) | 繁體中文 | [简体中文](README.zh-CN.md)

## 簡介

Claude Code 是 Anthropic 官方的 CLI 工具，提供 AI 編程助手和互動式開發支援。此安裝程式提供自動化設定，具有智慧網路偵測和可選的 JWT 認證功能。

---

## 認證選項

### 選項 1：JWT Token 認證（推薦）
安裝程式可自動取得和設定 JWT token，實現無縫認證：

1. 執行安裝程式
2. 當提示時，選擇 "y" 進行 JWT token 設定
3. 輸入您的 MediaTek 憑證
4. 安裝程式將自動：
   - 偵測最佳可用的 MLOP 端點
   - 取得 JWT token
   - 設定認證標頭

### 選項 2：手動 API 金鑰設定
如果您偏好手動設定或遇到 JWT 認證問題：

1. 前往 [MediaTek MLOP Gateway for OA](https://mlop-azure-gateway.mediatek.inc/auth/login) / [MediaTek MLOP Gateway for SWRD](https://mlop-azure-rddmz.mediatek.inc/auth/login)登入
2. 取得您的 GAISF API 金鑰
3. 在設定中手動設定金鑰

**注意**：安裝程式自動偵測網路連線性，並選擇 HTTP/HTTPS 協定以獲得最佳相容性。

## 安裝特性

安裝程式包含可靠設定的進階功能：

### 智慧依賴管理
- **Node.js 22+ 偵測**：自動檢查並安裝所需的 Node.js 版本
- **平台特定安裝**：
  - **macOS**：使用 Homebrew 自動安裝
  - **Linux**：支援多種套件管理器（apt、dnf、yum、pacman）
  - **Windows**：提供直接下載連結和引導安裝

### 智慧網路偵測
- **多註冊表支援**：自動測試 MediaTek 內部 npm 註冊表以獲得最佳下載速度
- **端點自動選擇**：偵測最佳可用的 MLOP 閘道端點
- **回退機制**：如果主要連線失敗，無縫切換到備用伺服器

### 設定管理
- **系統層級安裝**：支援使用者層級和系統層級設定
- **多平台二進位檔案**：安裝具有正確命名的平台特定 claude_analysis 二進位檔案
- **託管設定**：自動產生最佳化的 settings.json，支援遙測和 MCP 伺服器

---

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

### 第三步：設定認證
安裝過程中，您將被提示設定認證：

1. **JWT Token 設定（推薦）**：
   - 當詢問 JWT token 設定時選擇 "y"
   - 輸入您的 MediaTek 使用者名稱和密碼
   - 安裝程式將安全取得並設定您的 JWT token

2. **跳過 JWT 設定**：
   - 選擇 "N" 跳過 JWT 設定
   - 如需要，您可以稍後手動設定 API 金鑰

安裝程式自動處理所有技術設定，包括：
- 網路端點偵測和選擇
- 註冊表回退設定
- 平台特定的二進位安裝
- 最佳化的 Claude Code 設定

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
- **Node.js**：版本 22 或更高（在 macOS/Linux 上自動安裝）
- **網路連線**：下載元件和認證時需要
- **憑證**：MediaTek 帳戶用於 JWT 認證（可選但推薦）

## 安裝位置

安裝程式建立以下檔案和目錄：

### 使用者層級安裝（預設）
- **Claude CLI**：透過 npm 全域安裝
- **設定**：`~/.claude/settings.json`
- **二進位檔案**：`~/.claude/claude_analysis-{平台}-{架構}[.exe]`

### 系統層級安裝（如果可用）
- **macOS**：`/Library/Application Support/ClaudeCode/`
- **Linux**：`/etc/claude-code/`
- **Windows**：`C:\ProgramData\ClaudeCode\`

安裝程式根據系統權限和需求自動選擇最佳安裝方法。

## 疑難排解

**問題**：安裝後 `claude --version` 無法運作
**解決方案**：重新啟動終端機或將 npm 的全域 bin 目錄加入 PATH

**問題**：安裝過程中 JWT 認證失敗
**解決方案**：
- 驗證您的 MediaTek 憑證是否正確
- 檢查到 MLOP 閘道的網路連線
- 跳過 JWT 設定並根據需要手動設定

**問題**：Linux/macOS 上 Node.js 安裝失敗
**解決方案**：
- 安裝程式自動嘗試多個套件管理器
- 如果全部失敗，從 https://nodejs.org/ 手動安裝 Node.js 22+
- 手動安裝 Node.js 後重新執行安裝程式

**問題**：網路錯誤導致安裝失敗
**解決方案**：
- 安裝程式自動嘗試備用註冊表和端點
- 確保您有網路連線
- 在企業防火牆後嘗試使用適當的代理設定執行

**問題**：需要重新安裝或更新
**解決方案**：您可以安全地多次執行安裝程式 - 它會偵測現有安裝並適當更新

**問題**：設定檔未被讀取
**解決方案**：安裝程式在多個位置建立設定。檢查：
- `~/.claude/settings.json`（使用者層級）
- 系統層級託管設定（特定於作業系統的路徑）
- 參考 [官方設定文件](https://docs.anthropic.com/zh-TW/docs/claude-code/settings#%E8%A8%AD%E5%AE%9A%E6%AA%94%E6%A1%88) 了解其他位置

## 額外資源

- [官方文件](https://docs.anthropic.com/zh-TW/docs/claude-code)
- [設定文件](https://docs.anthropic.com/zh-TW/docs/claude-code/settings)
- [子代理功能](https://docs.anthropic.com/zh-TW/docs/claude-code/sub-agents) - 探索專業任務的代理功能
- [MCP 整合](https://docs.anthropic.com/zh-TW/docs/claude-code/mcp) - 了解模型上下文協定支援

## 取得協助

如果遇到問題：
1. 確保有網路連線
2. 嘗試以系統管理員身分執行安裝程式（Windows）或使用 `sudo`（macOS/Linux）進行系統層級安裝
3. 使用 `node --version` 檢查 Node.js 是否正確安裝（應為 22+）
4. 對於 JWT 認證問題，驗證您的 MediaTek 帳戶憑證
5. 檢查安裝程式輸出的特定錯誤訊息，如需要可使用不同選項重試

## 此版本的新功能

### 增強的認證
- **自動 JWT Token 管理**：安全的憑證處理，自動 token 重新整理
- **智慧端點偵測**：自動選擇最佳可用的 MLOP 閘道
- **憑證安全性**：在 Unix 系統上隱藏密碼輸入以確保安全

### 改進的網路可靠性
- **多註冊表支援**：MediaTek 內部 npm 註冊表之間的自動回退
- **連線性測試**：預安裝網路檢查確保最佳下載路徑
- **協定自動選擇**：基於網路條件的智慧 HTTP/HTTPS 選擇

### 進階安裝選項
- **平台特定二進位檔案**：為每個平台和架構正確命名的二進位檔案
- **系統層級設定**：支援企業層級託管設定
- **依賴項目自動安裝**：在所有支援的平台上自動化 Node.js 設定

### 設定增強
- **最佳化的預設設定**：預設定遙測、MCP 伺服器和協作功能
- **彈性的設定路徑**：支援使用者和系統層級設定檔
- **互動式設定**：使用者友善的認證和設定選擇提示

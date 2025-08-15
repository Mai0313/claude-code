# Claude Code 安裝程式

一個簡單的安裝工具，協助您設定 Claude Code 和開發活動分析功能。

## 這個安裝程式能做什麼？

安裝程式會自動：
1. **安裝 Node.js**（如果尚未安裝）- Claude Code 執行所需
2. **安裝 Claude Code CLI** - 主要應用程式
3. **設定分析整合** - 追蹤您的程式撰寫活動以獲得深入分析
4. **自動設定一切** - 無需手動設定

## 使用方法

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
- 分析元件已準備好追蹤您的開發活動

## 疑難排解

**問題**：安裝後 `claude --version` 無法運作
**解決方案**：重新啟動終端機或將 npm 的全域 bin 目錄加入 PATH

**問題**：第一次安裝失敗
**解決方案**：安裝程式會自動使用備用伺服器重試 - 請等待完成

**問題**：需要重新安裝或更新
**解決方案**：您可以安全地多次執行安裝程式 - 不會損壞任何東西

## 取得協助

如果遇到問題：
1. 確保有網路連線
2. 嘗試以系統管理員身分執行安裝程式（Windows）或使用 `sudo`（macOS/Linux）進行系統層級安裝
3. 使用 `node --version` 檢查 Node.js 是否正確安裝

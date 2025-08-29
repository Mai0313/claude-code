# Claude Analysis（Claude 分析工具）

一個遙測工具，用於收集和分析您的 Claude Code 開發活動，提供程式撰寫模式和生產力的深入分析。

## 這個工具能做什麼？

Claude Analysis 自動：
1. **追蹤您的程式撰寫活動** - 監控檔案讀取、寫入和程式碼變更
2. **分析開發模式** - 統計撰寫的行數、處理的字元數和工具使用情況
3. **傳送分析資料** - 將彙整統計資料上傳到遙測伺服器以獲得洞察
4. **產生使用報告** - 回傳關於您開發會話的結構化資料

## 運作原理

該工具以 STOP 模式操作：

### STOP 模式（預設）
- 從標準輸入讀取包含 `transcript_path` 的 Python 字典
- 載入並解析 JSONL 對話記錄檔案
- 彙整會話中的所有開發活動
- 將分析資料傳送到遙測伺服器

## 使用方法

### 基本用法
```bash
# STOP 模式（預設）- 從標準輸入讀取記錄路徑
echo "{'transcript_path': '/path/to/conversation.jsonl'}" | ./claude_analysis

# 直接檔案分析模式 - 直接分析 JSONL 檔案
./claude_analysis --path examples/test_conversation.jsonl

# 直接檔案分析並輸出到檔案
./claude_analysis --path examples/test_conversation.jsonl --output analysis.json

# 自訂 API 端點與 stdin 模式
./claude_analysis --o11y_base_url https://custom-server.com/api/upload < input.json
```

### 命令列選項
- `--path`: 直接分析的 JSONL 檔案路徑（選填，替代 stdin 模式）
- `--output`: 儲存分析結果的 JSON 檔案路徑（選填，預設：stdout）
- `--o11y_base_url`: 覆蓋預設的 API 端點 URL（預設值：`https://gaia.mediatek.inc/o11y/upload_locs`）
- `--check-update`: 檢查可用更新並結束
- `--skip-update-check`: 跳過啟動時的自動更新檢查
- `--version`: 顯示版本資訊並結束

### 使用模式

#### 1. 傳統 STOP 模式（預設）
```bash
# 從 stdin 讀取，傳送到 API
echo "{'transcript_path': '/path/to/conversation.jsonl'}" | ./claude_analysis
```
- 輸入：從 stdin 接收包含 `transcript_path` 的 JSON
- 輸出：JSON 格式的 API 回應
- 行為：載入 JSONL → 分析 → 傳送到 API

#### 2. 直接檔案分析模式
```bash
# 分析檔案並輸出到 stdout
./claude_analysis --path examples/test_conversation.jsonl

# 分析檔案並儲存到 JSON 檔案
./claude_analysis --path examples/test_conversation.jsonl --output result.json
```
- 輸入：透過 `--path` 直接指定 JSONL 檔案路徑
- 輸出：分析結果到 stdout 或檔案（透過 `--output`）
- 行為：載入 JSONL → 分析 → 輸出 JSON（無 API 呼叫）

### 環境變數
- 也可以在工作目錄中建立包含環境設定的 `.env` 檔案

### 輸入格式

**STOP 模式輸入：**
```
{'transcript_path': '/absolute/path/to/conversation.jsonl'}
```

**直接分析模式：**
```bash
# 使用 --path 參數（不需要 stdin）
./claude_analysis --path /absolute/path/to/conversation.jsonl

# 帶自訂輸出位置
./claude_analysis --path /absolute/path/to/conversation.jsonl --output /path/to/output.json
```

## 追蹤什麼內容？

該工具分析和報告：

### 檔案操作
- **讀取操作**：開啟和讀取的檔案內容
- **寫入操作**：建立或修改的檔案
- **差異操作**：套用的程式碼修補和變更

### 產生的統計資料
- 存取的唯一檔案總數
- 撰寫的總行數
- 讀取/寫入/修改的總字元數
- 工具使用次數（Read、Write、ApplyDiff 等）
- 會話中繼資料（工作區路徑、git 儲存庫、時間戳記）

### 輸出格式

- [Example Output](./examples/claude_code_log.json)

```json
{
  "user": "your-username",
  "records": [{
    "totalUniqueFiles": 5,
    "totalWriteLines": 120,
    "totalReadCharacters": 2500,
    "totalWriteCharacters": 1800,
    "totalDiffCharacters": 350,
    "toolCallCounts": {"Read": 8, "Write": 3, "ApplyDiff": 1},
    "taskId": "session-id",
    "timestamp": 1704067200000,
    "folderPath": "/path/to/workspace",
    "gitRemoteUrl": "https://github.com/user/repo.git"
  }],
  "extensionName": "Claude-Code",
  "machineId": "unique-machine-id",
  "insightsVersion": "0.0.1"
}
```

## 設定

工具使用這些預設設定：
- **API 端點**：`https://gaia.mediatek.inc/o11y/upload_locs`（可透過 `--o11y_base_url` 覆蓋）
- **逾時時間**：10 秒
- **擴充套件名稱**："Claude-Code"
- **洞察版本**："0.0.1"

大部分設定會自動從您的系統載入（使用者名稱、機器 ID）。API 端點可以透過 `--o11y_base_url` 命令列選項進行自訂。

## 自動更新功能

Claude Analysis 包含自動更新檢查功能，透過 Gitea API 檢查新版本的可用性。

### 自動更新選項

| 命令 | 描述 |
|------|------|
| `--check-update` | 手動檢查更新並顯示最新版本資訊 |
| `--skip-update-check` | 跳過啟動時的自動更新檢查 |

### 使用範例

```bash
# 手動檢查更新
./claude_analysis --check-update

# 跳過自動更新檢查執行
./claude_analysis --skip-update-check

# 檢視目前版本
./claude_analysis --version
```

### 環境變數設定

- `CLAUDE_ANALYSIS_SKIP_UPDATE`: 設定為 `true` 可全域停用自動更新檢查
- `CLAUDE_ANALYSIS_UPDATE_URL`: 自訂更新檢查的 API 端點（預設：Gitea API）

### 更新行為

- **自動檢查**：工具啟動時會檢查新版本（可透過 `--skip-update-check` 或環境變數停用）
- **跨平台支援**：
  - Linux/macOS：支援自動更新檢查和通知
  - Windows：僅顯示手動更新通知
- **優雅處理**：更新檢查失敗不會影響工具的正常功能
- **版本比較**：使用語意版本控制比較目前版本與最新可用版本

### 更新通知

當偵測到新版本時，工具會顯示類似以下的通知：

```
🚀 新版本可用！
目前版本: v1.0.0
最新版本: v1.1.0
請造訪：https://gitea.mediatek.inc/IT-GAIA/claude-code/releases 下載最新版本
```

## 整合

此工具通常用作 Claude Code 中的掛鉤：
1. Claude Code 產生對話記錄
2. 記錄路徑傳遞給 claude_analysis
3. 分析資料被處理並傳送到遙測伺服器
4. 回傳結果供進一步處理

## 疑難排解

**問題**：工具無法讀取記錄檔案
**解決方案**：確保輸入中的記錄路徑是絕對路徑且檔案存在

**問題**：網路逾時錯誤（僅 STOP 模式）
**解決方案**：檢查您的網路連線和遙測端點的防火牆設定

**問題**：JSON 解析錯誤
**解決方案**：驗證您的輸入格式是否符合所選模式的預期結構

**問題**：空輸出
**解決方案**：檢查您的記錄檔案是否包含帶有工具使用事件的有效對話資料

**問題**：檔案未找到錯誤（直接模式）
**解決方案**：驗證透過 `--path` 指定的路徑存在且可存取

**問題**：寫入輸出檔案時權限被拒絕
**解決方案**：確保 `--output` 的目錄存在且您有寫入權限

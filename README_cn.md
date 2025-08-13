# Claude Analysis - Go 版本

這是將原本的 Python `post_hook.py` 移植為 Go 的 `claude_analysis` 專案，可跨平台編譯為單一執行檔。

## 功能

- 從標準輸入讀取 JSON
- 取得當前使用者資訊
- 將資料送往 HTTP API 端點並回傳結果
- 支援跨平台編譯
- 解析 JSONL 對話紀錄並彙整為統計資料（聚合 Parser）

## 使用方式

```bash
# 建置
make build

# 基本測試
echo '{"key": "value"}' | ./build/claude_analysis

# Mode = STOP（預設）：stdin 輸入 Python 字典含 transcript_path，程式讀檔並彙整
echo "{'transcript_path':'/絕對路徑/tests/test_conversation.jsonl'}" | ./build/claude_analysis

# Mode = POST_TOOL：stdin 直接輸入一行或多行 JSON（對話中的任意子集合），程式直接彙整
MODE=POST_TOOL ./build/claude_analysis <<'EOF'
{"type":"assistant","uuid":"u1","cwd":"/tmp/ws","sessionId":"s1","timestamp":"2025-01-01T00:00:00Z","message":{"content":[{"type":"tool_use","name":"Read"}]}}
{"parentUuid":"u1","timestamp":"2025-01-01T00:00:01Z","toolUseResult":{"filePath":"a.txt","content":"hello"}}
EOF
```

## 聚合 Parser 說明

- 呼叫順序：`telemetry.ReadJSONL(path)` → `telemetry.AggregateConversationStats(records)`
- 輸出結構會放在請求 payload 的 `records` 欄位中，包含：
  - `totalUniqueFiles`、`totalWriteLines`、`totalReadCharacters`、`totalWriteCharacters`、`totalDiffCharacters`
  - `writeToFileDetails`、`readFileDetails`、`applyDiffDetails`
  - `toolCallCounts`、`taskId`、`timestamp`、`folderPath`、`gitRemoteUrl`
  - `user` 透過系統使用者取得，`machineId` 透過系統 machine id 取得，`extensionName` 固定為 `Claude-Code`，`insightsVersion` 固定為 `v0.0.1`
- 會嘗試從 `cwd/.git/config` 讀取 `remote.origin.url` 作為 `gitRemoteUrl`（盡力而為）

## 建置

```bash
make build
# 或直接
mkdir -p build && go build -o build/claude_analysis ./cmd/claude_analysis
```

## 其他

- API 端點、逾時等設定可在 `core/config/config.go` 中調整
- 專案使用 Go 標準函式庫，不需額外依賴

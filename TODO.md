請看一下我現在的專案
目前做法是 先透過stdin來獲取一個dict
從中間取得TranscriptPath
再透過這個path去讀取jsonl

我現在的問題是 你能不能幫我寫一個parser 將 `./tests/test_conversation.jsonl` parse成下面這種格式
勁量不要動到主邏輯 我希望透過這個parser 直接接在 `data, err := telemetry.ReadJSONL(filepath)` 後面
```json
[
  {
    "user": "ds906659",
    "records": [
      {
        "totalUniqueFiles": 2,
        "totalWriteLines": 48,
        "totalReadCharacters": 12243,
        "totalWriteCharacters": 1516,
        "totalDiffCharacters": 12115,
        "writeToFileDetails": [
          {
            "filePath": "mcp.md",
            "lineCount": 48,
            "characterCount": 1516,
            "timestamp": 1750405776513,
            "aiOutputContent": ".....",
            "fileContent": "...",
          }
        ],
        "readFileDetails": [
          {
            "filePath": "main.py",
            "characterCount": 12243,
            "timestamp": 1750405054609,
            "aiOutputContent": "...",
            "fileContent": "...",
          }
        ],
        ...
      }
    ]
  }
]
```
需要轉換成哪些類別你能參考下面的typescript範例
```typescript
export interface WriteToFileDetail {
  filePath: string;
  lineCount: number;
  characterCount: number;
  timestamp: number;
  aiOutputContent: string;
  fileContent: string;
}

export interface ReadFileDetail {
  filePath: string;
  characterCount: number;
  timestamp: number;
  aiOutputContent: string;
  fileContent: string;
}

export interface ApplyDiffDetail {
  filePath: string;
  characterCount: number;
  timestamp: number;
  aiOutputContent: string;
  fileContent: string;
}

export interface ApiConversationStats {
  totalUniqueFiles: number; // 涉及的唯一文件數量
  totalWriteLines: number; // 寫入的總行數
  totalReadCharacters: number; // 讀取文件的總字符數
  totalWriteCharacters: number; // 寫入文件的總字符數
  totalDiffCharacters: number; // apply_diff 修改的總字符數
  writeToFileDetails: WriteToFileDetail[];
  readFileDetails: ReadFileDetail[];
  applyDiffDetails: ApplyDiffDetail[];
  toolCallCounts: Record<string, number>;
  taskId: string;
  timestamp: number;
  folderPath: string; // 工作空間絕對路徑
  gitRemoteUrl: string; // Git remote origin URL
}

export interface ApiConversationAnalysis {
  user: string;
  records: ApiConversationStats[];
  extensionName: string;
  machineId: string;
  insightsVersion: string;
}
```

# 專案結構說明

## 目錄結構

```
claude_analysis/
├── cmd/                        # 主要的應用程式進入點
│   └── claude_analysis/        # 主程式
│       └── main.go            # 程式入口點
├── internal/                   # 私有的應用程式程式碼（不會被其他專案匯入）
│   ├── config/                # 配置管理
│   │   └── config.go          # 應用程式配置
│   └── telemetry/             # 遙測功能
│       ├── client.go          # HTTP 客戶端實作
│       └── input.go           # 標準輸入處理
├── pkg/                       # 公共的函式庫程式碼（可被其他專案匯入）
├── build/                     # 建置輸出目錄
├── docs/                      # 專案文檔
│   └── project_structure.md   # 本檔案
├── scripts/                   # 建置、安裝、分析等腳本
├── go.mod                     # Go 模組定義
├── Makefile                   # 建置腳本
└── README.md                  # 專案說明
```

## 模組說明

### cmd/ 目錄
- 存放主要的應用程式進入點
- 每個子目錄代表一個可執行的程式
- `cmd/claude_analysis/main.go` 是主程式的進入點

### internal/ 目錄
- 存放私有的應用程式程式碼
- 這些程式碼不能被其他 Go 專案匯入
- **config/**: 處理應用程式配置
- **telemetry/**: 處理遙測數據的提交和標準輸入讀取

### pkg/ 目錄
- 存放可以被其他專案匯入的公共函式庫
- 目前是空的，但為未來擴展預留

### docs/ 目錄
- 存放專案相關的文檔

### scripts/ 目錄
- 存放各種腳本，如建置、部署、測試腳本等

## 設計原則

1. **單一職責**: 每個模組都有清楚的職責分工
2. **可測試性**: 將業務邏輯分離到 internal 包中，方便進行單元測試
3. **可維護性**: 模組化的結構讓程式碼更容易維護和擴展
4. **Go 慣例**: 遵循 Go 社群的標準專案結構

## 建置方法

```bash
# 建置當前平台
make build

# 建置所有平台
make build-all

# 執行程式
echo '{"test": "data"}' | make run
```

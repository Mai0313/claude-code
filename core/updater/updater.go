package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"claude_analysis/core/version"
)

const (
	// GitEA API 基础配置
	GITEA_BASE_URL = "https://gitea.mediatek.inc"
	REPO_OWNER     = "IT-GAIA"
	REPO_NAME      = "claude-code"
	RELEASES_API   = "/api/v1/repos/IT-GAIA/claude-code/releases"

	// 更新配置
	CHECK_TIMEOUT = 10 * time.Second
)

// Release 表示 Gitea API 返回的 release 信息
type Release struct {
	ID          int64     `json:"id"`
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	CreatedAt   time.Time `json:"created_at"`
	PublishedAt time.Time `json:"published_at"`
}

// UpdateResult 表示更新操作的結果
type UpdateResult struct {
	HasUpdate      bool   `json:"has_update"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	Message        string `json:"message"`
	Error          string `json:"error,omitempty"`
}

// CheckForUpdatesGraceful 檢查是否有新版本可用，優雅處理錯誤
func CheckForUpdatesGraceful() (*UpdateResult, error) {
	result := &UpdateResult{
		CurrentVersion: version.GetVersion(),
	}

	// 獲取最新版本信息
	latestRelease, err := getLatestRelease()
	if err != nil {
		result.Error = fmt.Sprintf("Failed to get latest release: %v", err)
		result.Message = "Unable to check for updates, continuing with current version"
		return result, err
	}

	result.LatestVersion = latestRelease.TagName

	// 比較版本
	if !version.IsNewerVersion(result.CurrentVersion, result.LatestVersion) {
		result.Message = "Already using the latest version"
		return result, nil
	}

	result.HasUpdate = true
	result.Message = fmt.Sprintf("New version available: %s -> %s", result.CurrentVersion, result.LatestVersion)
	return result, nil
}

// getLatestRelease 從 Gitea API 獲取最新的 release
func getLatestRelease() (*Release, error) {
	client := &http.Client{Timeout: CHECK_TIMEOUT}

	url := fmt.Sprintf("%s%s?draft=false&pre-release=false", GITEA_BASE_URL, RELEASES_API)
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found")
	}

	return &releases[0], nil
}

// ForceUpdateCheck 友善地檢查更新並提醒用戶，不會中斷程序執行
func ForceUpdateCheck() error {
	result, err := CheckForUpdatesGraceful()
	if err != nil {
		// 如果無法檢查更新，僅記錄警告但不阻止程序運行
		fmt.Fprintf(os.Stderr, "\n⚠️  無法檢查更新: %v\n", err)
		fmt.Fprintf(os.Stderr, "請稍後手動檢查更新: %s/%s/%s/releases\n\n", GITEA_BASE_URL, REPO_OWNER, REPO_NAME)
		return nil
	}

	if result.HasUpdate {
		fmt.Fprintf(os.Stderr, "\n🔔 有新版本可用！\n")
		fmt.Fprintf(os.Stderr, "當前版本: %s\n", result.CurrentVersion)
		fmt.Fprintf(os.Stderr, "最新版本: %s\n", result.LatestVersion)
		fmt.Fprintf(os.Stderr, "\n建議手動更新：\n")
		fmt.Fprintf(os.Stderr, "  前往下載頁面: %s/%s/%s/releases\n", GITEA_BASE_URL, REPO_OWNER, REPO_NAME)
		fmt.Fprintf(os.Stderr, "\n程序將在 3 秒後繼續執行...\n\n")

		// 等待 3 秒讓用戶看到提示
		time.Sleep(3 * time.Second)
	}

	return nil
}

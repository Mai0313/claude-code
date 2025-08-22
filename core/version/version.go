package version

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
)

// These variables will be set at build time via -ldflags
var (
	// Version is the current version of the application
	Version = "dev"
	// BuildTime is when the binary was built
	BuildTime = "unknown"
	// GitCommit is the git commit hash
	GitCommit = "unknown"
)

// Info holds version information
type Info struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GitCommit string `json:"git_commit"`
	GoVersion string `json:"go_version"`
}

// SemanticVersion 表示語義化版本
type SemanticVersion struct {
	Major int
	Minor int
	Patch int
	Pre   string // 預發布版本標識符 (如 "alpha.1", "beta.2")
	Build string // 構建元數據 (如 "dirty", "20220101.abcdef")
}

// Get returns version information
func Get() Info {
	info := Info{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
		GoVersion: getGoVersion(),
	}

	// If version is still "dev", try to get it from build info (for go install)
	if info.Version == "dev" {
		if buildInfo, ok := debug.ReadBuildInfo(); ok {
			if buildInfo.Main.Version != "(devel)" && buildInfo.Main.Version != "" {
				info.Version = buildInfo.Main.Version
			}
		}
	}

	return info
}

// GetVersion returns just the version string
func GetVersion() string {
	return Get().Version
}

// IsNewerVersion 檢查是否有新版本可用（簡化版本）
func IsNewerVersion(current, latest string) bool {
	// 移除 'v' 前綴
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	// 如果當前版本是 "dev" 或 "unknown"，則認為需要更新
	if current == "dev" || current == "unknown" {
		return true
	}

	// 處理 "dirty" 版本（如 "0.2.3+dirty"）
	if strings.Contains(current, "+") {
		current = strings.Split(current, "+")[0]
	}

	// 如果版本相同，則不需要更新
	if current == latest {
		return false
	}

	// 嘗試語義化版本比較
	currentVer, err1 := ParseVersion(current)
	latestVer, err2 := ParseVersion(latest)

	if err1 == nil && err2 == nil {
		return latestVer.IsNewer(currentVer)
	}

	// 如果解析失敗，使用字符串比較
	return current != latest
}

// ParseVersion 解析語義化版本字符串
func ParseVersion(version string) (*SemanticVersion, error) {
	// 移除 'v' 前綴
	version = strings.TrimPrefix(version, "v")

	if version == "dev" || version == "unknown" || version == "" {
		return &SemanticVersion{0, 0, 0, "dev", ""}, nil
	}

	sv := &SemanticVersion{}

	// 分離構建元數據 (+ 後面的部分)
	if idx := strings.Index(version, "+"); idx != -1 {
		sv.Build = version[idx+1:]
		version = version[:idx]
	}

	// 分離預發布版本 (- 後面的部分)
	if idx := strings.Index(version, "-"); idx != -1 {
		sv.Pre = version[idx+1:]
		version = version[:idx]
	}

	// 解析主版本號
	parts := strings.Split(version, ".")
	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid version format: %s", version)
	}

	var err error
	if sv.Major, err = strconv.Atoi(parts[0]); err != nil {
		return nil, fmt.Errorf("invalid major version: %s", parts[0])
	}

	// 解析次版本號
	if len(parts) >= 2 {
		if sv.Minor, err = strconv.Atoi(parts[1]); err != nil {
			return nil, fmt.Errorf("invalid minor version: %s", parts[1])
		}
	}

	// 解析修訂版本號
	if len(parts) >= 3 {
		if sv.Patch, err = strconv.Atoi(parts[2]); err != nil {
			return nil, fmt.Errorf("invalid patch version: %s", parts[2])
		}
	}

	return sv, nil
}

// Compare 比較兩個版本
// 返回值: -1 (當前版本較小), 0 (相等), 1 (當前版本較大)
func (sv *SemanticVersion) Compare(other *SemanticVersion) int {
	// 比較主版本號
	if sv.Major != other.Major {
		if sv.Major < other.Major {
			return -1
		}
		return 1
	}

	// 比較次版本號
	if sv.Minor != other.Minor {
		if sv.Minor < other.Minor {
			return -1
		}
		return 1
	}

	// 比較修訂版本號
	if sv.Patch != other.Patch {
		if sv.Patch < other.Patch {
			return -1
		}
		return 1
	}

	// 比較預發布版本
	if sv.Pre == "" && other.Pre != "" {
		return 1 // 正式版本大於預發布版本
	}
	if sv.Pre != "" && other.Pre == "" {
		return -1 // 預發布版本小於正式版本
	}
	if sv.Pre != other.Pre {
		return strings.Compare(sv.Pre, other.Pre)
	}

	// 構建元數據不影響版本優先級
	return 0
}

// IsNewer 檢查當前版本是否比另一個版本新
func (sv *SemanticVersion) IsNewer(other *SemanticVersion) bool {
	return sv.Compare(other) > 0
}

func getGoVersion() string {
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		return buildInfo.GoVersion
	}
	return "unknown"
}

package version

import "runtime/debug"

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

func getGoVersion() string {
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		return buildInfo.GoVersion
	}
	return "unknown"
}

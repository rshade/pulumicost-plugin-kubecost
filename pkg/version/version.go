package version

import (
	"fmt"
	"runtime"
)

const (
	defaultVersion = "1.0.0"
	unknownValue   = "unknown"
)

// defaultVersionInfo provides default version information.
func defaultVersionInfo() (string, string, string, string, string) {
	// These values can be overridden at build time using ldflags
	return defaultVersion, unknownValue, unknownValue, unknownValue, unknownValue
}

// Info contains version information.
type Info struct {
	Version   string `json:"version"`
	BuildDate string `json:"buildDate"`
	GitCommit string `json:"gitCommit"`
	GitBranch string `json:"gitBranch"`
	GitState  string `json:"gitState"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}

// GetVersionInfo returns the complete version information.
func GetVersionInfo() Info {
	version, buildDate, gitCommit, gitBranch, gitState := defaultVersionInfo()
	return Info{
		Version:   version,
		BuildDate: buildDate,
		GitCommit: gitCommit,
		GitBranch: gitBranch,
		GitState:  gitState,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string.
func String() string {
	info := GetVersionInfo()
	return fmt.Sprintf("v%s (%s, %s, %s)",
		info.Version,
		info.GitCommit,
		info.BuildDate,
		info.Platform)
}

// FullString returns a detailed version string.
func FullString() string {
	info := GetVersionInfo()
	return fmt.Sprintf(
		"Version: %s\nBuild Date: %s\nGit Commit: %s\nGit Branch: %s\nGit State: %s\nGo Version: %s\nPlatform: %s",
		info.Version,
		info.BuildDate,
		info.GitCommit,
		info.GitBranch,
		info.GitState,
		info.GoVersion,
		info.Platform)
}

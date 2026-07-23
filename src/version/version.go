package version

// Package: version
// Author: Nikola Jurkovic
// License: GPL-3.0 or later

import (
	"runtime/debug"
	"time"
)

type BuildInfo struct {
	Revision     string    `json:"revision"`
	Time         time.Time `json:"time"`
	Modified     bool      `json:"modified"`
	BuildVersion string    `json:"buildVersion"`
}

// Version is the tracked development fallback. Authorized release builds may override it through -ldflags.
var Version = "0.2.0-alpha-dev"
var buildInfo *BuildInfo

// GetBuildInfo will return BuildInfo struct
func GetBuildInfo() *BuildInfo {
	return buildInfo
}

// Init will initialize a new version object
func Init() {
	buildInfo = getBuildInfo()
}

// shortSHA will shorten revision SHA
func shortSHA(sha string) string {
	const shortLen = 7
	if len(sha) >= shortLen {
		return sha[:shortLen]
	}
	return sha
}

// getBuildInfo will fetch the latest build info
func getBuildInfo() *BuildInfo {
	build := &BuildInfo{
		Revision:     "",
		Time:         time.Time{},
		Modified:     false,
		BuildVersion: Version,
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, kv := range info.Settings {
			switch kv.Key {
			case "vcs.revision":
				build.Revision = shortSHA(kv.Value)
			case "vcs.time":
				build.Time, _ = time.Parse(time.RFC3339, kv.Value)
			case "vcs.modified":
				build.Modified = kv.Value == "true"
			}
		}
	}
	return build
}

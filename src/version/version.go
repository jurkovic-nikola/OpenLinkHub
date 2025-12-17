package version

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

// Version is set via -ldflags automatically upon a build process, no need to modify this manually.
var Version = "0.0.0"
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

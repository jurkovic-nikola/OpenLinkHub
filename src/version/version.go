package version

import (
	"runtime/debug"
	"time"
)

type BuildInfo struct {
	Revision string    `json:"revision"`
	Time     time.Time `json:"time"`
	Modified bool      `json:"modified"`
}

var buildInfo *BuildInfo

// GetBuildInfo will return BuildInfo struct
func GetBuildInfo() *BuildInfo {
	return buildInfo
}

// Init will initialize a new version object
func Init() {
	buildInfo = getBuildInfo()
}

// getBuildInfo will fetch the latest build info
func getBuildInfo() *BuildInfo {
	build := &BuildInfo{
		Revision: "",
		Time:     time.Time{},
		Modified: false,
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, kv := range info.Settings {
			switch kv.Key {
			case "vcs.revision":
				build.Revision = kv.Value
			case "vcs.time":
				build.Time, _ = time.Parse(time.RFC3339, kv.Value)
			case "vcs.modified":
				build.Modified = kv.Value == "true"
			}
		}
	}
	return build
}

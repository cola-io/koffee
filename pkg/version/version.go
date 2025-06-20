package version

import (
	"encoding/json"
	"fmt"
	"runtime"
)

var (
	// NOTE: The $Format strings are replaced during 'git archive' thanks to the
	// companion .gitattributes file containing 'export-subst' in this same
	// directory.  See also https://git-scm.com/docs/gitattributes
	module    = "unknown"
	version   = "v0.0.0-master+$Format:%h$"
	gitCommit = "$Format:%H$"          // sha1 from git, output of $(git rev-parse HEAD)
	buildDate = "1970-01-01T00:00:00Z" // build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
)

// Info contains versioning information.
type Info struct {
	Module    string `json:"module"`
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
	Compiler  string `json:"compiler"`
	Platform  string `json:"platform"`
}

// Pretty returns a pretty output representation of Info
func (info Info) Pretty() string {
	return fmt.Sprintf(
		"Module: %s\nVersion: %s\nGitCommit: %s\nBuildDate: %s\nGoVersion: %s\nPlatform: %s",
		info.Module,
		info.Version,
		info.GitCommit,
		info.BuildDate,
		info.GoVersion,
		info.Platform,
	)
}

func (info Info) Short() string {
	return fmt.Sprintf("%s-%s", info.Version, info.GitCommit)
}

// String returns the marshalled json string of Info
func (info Info) String() string {
	str, _ := json.Marshal(info)
	return string(str)
}

// Get returns the overall codebase version. It's for detecting
// what code a binary was built from.
func Get() Info {
	// These variables typically come from -ldflags settings and in
	// their absence fallback to the settings in version/base.go
	return Info{
		Module:    module,
		Version:   version,
		GitCommit: gitCommit,
		BuildDate: buildDate,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

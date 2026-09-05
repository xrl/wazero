package version

import (
	"runtime/debug"
)

// Default is the default version value used when none was found.
const Default = "dev"

// version holds the current version from the go.mod of downstream users or set by ldflag for wazero CLI.
var version string

// GetWazeroVersion returns the current version of wazero either in the go.mod or set by ldflag for wazero CLI.
//
// If this is not CLI, this assumes that downstream users of wazero imports wazero as "github.com/tetratelabs/wazero".
// A replacement module takes precedence over the require statement.
// For example, if the go.mod has "require github.com/tetratelabs/wazero 0.1.2-12314124-abcd",
// then this returns "0.1.2-12314124-abcd".
//
// Note: this is tested in ./testdata/main_test.go with a separate go.mod to pretend as the wazero user.
func GetWazeroVersion() (ret string) {
	if len(version) != 0 {
		return version
	}

	info, ok := debug.ReadBuildInfo()
	if ok {
		ret = buildInfoVersion(info)
	}
	if versionMissing(ret) {
		return Default // don't return parens
	}

	// Cache for the subsequent calls.
	version = ret
	return ret
}

func versionMissing(ret string) bool {
	return ret == "" || ret == "(devel)" // pkg.go defaults to (devel)
}

// buildInfoVersion uses the actual replacement, never the replaced upstream
// version. Local replacements have no reproducible version and remain dev.
func buildInfoVersion(info *debug.BuildInfo) string {
	for _, dep := range info.Deps {
		if dep.Path == "github.com/tetratelabs/wazero" {
			if dep.Replace != nil {
				dep = dep.Replace
			}
			if versionMissing(dep.Version) {
				return Default
			}
			return dep.Version
		}
	}
	if versionMissing(info.Main.Version) {
		return Default
	}
	return info.Main.Version
}

package version

import (
	"runtime/debug"
)

var (
	// Version is filled via Makefile
	Version = ""
	// Revision is filled via Makefile
	Revision = ""
)

const unknown = "<unknown>"

func GetVersion() string {
	if Version != "" {
		return Version
	}

	if bi, ok := debug.ReadBuildInfo(); ok {
		if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			return bi.Main.Version
		}
	}
	return unknown
}

package info

import "runtime/debug"

var (
	// Version is the version app.
	Version = ""
)

func init() {
	if Version != "" {
		return
	}

	// If not set, get the information from the runtime in case Sloth has been used as a library.
	info, ok := debug.ReadBuildInfo()
	if ok {
		// Search for sloth as a library.
		for _, d := range info.Deps {
			if d.Path == "github.com/slok/sloth" {
				Version = d.Version
				return
			}
		}
	}

	// If still not set, then set to dev.
	Version = "dev"
}

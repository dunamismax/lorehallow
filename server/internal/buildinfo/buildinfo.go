// Package buildinfo holds the build version and git commit, injected at
// link time with -ldflags, plus the protocol version the server speaks.
package buildinfo

import "github.com/dunamismax/lorehallow/server/internal/pb"

// Set with -ldflags "-X github.com/dunamismax/lorehallow/server/internal/buildinfo.Version=..."
var (
	// Version is the game build version, for example v0.2.0 or a git describe string.
	Version = "dev"
	// Commit is the full git commit hash the binary was built from.
	Commit = "unknown"
)

// Info is the JSON shape served by GET /version.
type Info struct {
	Version         string `json:"version"`
	Commit          string `json:"commit"`
	ProtocolVersion int    `json:"protocol_version"`
}

// Get returns the build information for this binary.
func Get() Info {
	return Info{Version: Version, Commit: Commit, ProtocolVersion: pb.ProtocolVersion}
}

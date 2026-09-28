// Package version exposes the build version shared by the cantcp binaries.
package version

// Version is the application version reported by --version. Release builds
// override it with the linker:
//
//	go build -ldflags "-X github.com/burn-lab-dev/cantcp/internal/version.Version=v0.1.0"
var Version = "dev"

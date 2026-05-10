package version

// Build-time injected version information
// These values are set via ldflags during build:
// -ldflags "-X internal/version.Version=x.x.x -X internal/version.Commit=abc123 -X internal/version.BuildTime=2024-01-01"

var (
	Version   = "dev"       // Version from release tag
	Commit    = "unknown"   // Git commit hash
	BuildTime = "unknown"   // Build timestamp
	GoVersion = "unknown"   // Go compiler version
)

// GetVersionInfo returns all version information as a map
func GetVersionInfo() map[string]string {
	return map[string]string{
		"version":     Version,
		"commit":      Commit,
		"build_time":  BuildTime,
		"go_version":  GoVersion,
	}
}

// GetVersion returns just the version string
func GetVersion() string {
	return Version
}

// GetFullVersion returns version with commit info
func GetFullVersion() string {
	return Version + "-" + Commit
}
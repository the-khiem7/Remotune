//go:build windows

package lifecycle

import (
	"os"
	"path/filepath"
)

// CheckPortablePath verifies whether the currently running executable matches the
// autostart registration. If the executable has moved (e.g. a portable .exe was
// relocated), the autostart entry becomes stale and will silently fail at next
// logon. This function returns a diagnostic that the UI or tray can surface.
type PortablePathStatus struct {
	// AutostartRegistered is true if a Start with Windows entry exists.
	AutostartRegistered bool
	// PathMismatch is true when the autostart path does not match the running exe.
	// This means "Start with Windows" will fail silently after the next reboot.
	PathMismatch bool
	// RegisteredPath is the stale path from the registry (empty if not registered).
	RegisteredPath string
	// CurrentPath is the running executable's resolved path.
	CurrentPath string
}

// CheckPortablePath detects a moved-portable-path condition.
func CheckPortablePath() PortablePathStatus {
	status, err := GetAutostartStatus()
	if err != nil || !status.Registered {
		return PortablePathStatus{AutostartRegistered: false}
	}

	exe, _ := os.Executable()
	exe = filepath.Clean(exe)

	return PortablePathStatus{
		AutostartRegistered: true,
		PathMismatch:        !status.PathMatch,
		RegisteredPath:      status.RegisteredPath,
		CurrentPath:         exe,
	}
}

// RepairAutostartPath updates the autostart registration to point to the current
// executable path, fixing a moved-portable-path situation.
func RepairAutostartPath() error {
	return SetAutostart(true)
}

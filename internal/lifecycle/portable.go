//go:build windows

package lifecycle

import (
	"os"
	"path/filepath"
)

type PortablePathStatus struct {
	AutostartRegistered bool
	PathMismatch        bool
	RegisteredPath      string
	CurrentPath         string
}

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
func RepairAutostartPath() error {
	return SetAutostart(true)
}

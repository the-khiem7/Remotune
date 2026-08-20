//go:build windows

package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// autostartKeyPath is the standard Run key for per-user startup programs.
const autostartKeyPath = `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`
const autostartValueName = "Remotune"

// AutostartStatus reports whether "Start with Windows" is currently registered and
// whether the registered path matches the running executable.
type AutostartStatus struct {
	Registered bool
	PathMatch  bool
	// RegisteredPath is the path stored in the registry, if any.
	RegisteredPath string
}

// GetAutostartStatus reads the current autostart state from the registry.
func GetAutostartStatus() (AutostartStatus, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, autostartKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return AutostartStatus{}, fmt.Errorf("open Run key: %w", err)
	}
	defer k.Close()

	val, _, err := k.GetStringValue(autostartValueName)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return AutostartStatus{Registered: false}, nil
		}
		return AutostartStatus{}, fmt.Errorf("read autostart value: %w", err)
	}

	exe, _ := os.Executable()
	exe = normalizeAutostartPath(exe)
	registeredClean := normalizeAutostartPath(val)

	return AutostartStatus{
		Registered:     true,
		PathMatch:      equalsIgnoreCase(exe, registeredClean),
		RegisteredPath: val,
	}, nil
}

// SetAutostart enables or disables autostart using the current executable path.
// If the executable has moved since the last registration, enabling will update the
// path. Disabling always succeeds (even if not currently registered).
func SetAutostart(enable bool) error {
	if !enable {
		return removeAutostart()
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	exe = filepath.Clean(exe)

	k, err := registry.OpenKey(registry.CURRENT_USER, autostartKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open Run key for write: %w", err)
	}
	defer k.Close()

	// Quote the path for safety, even though paths without spaces technically don't need it.
	quoted := `"` + exe + `"`
	if err := k.SetStringValue(autostartValueName, quoted); err != nil {
		return fmt.Errorf("write autostart value: %w", err)
	}
	return nil
}

func removeAutostart() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, autostartKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open Run key for write: %w", err)
	}
	defer k.Close()

	err = k.DeleteValue(autostartValueName)
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("delete autostart value: %w", err)
	}
	return nil
}

// equalsIgnoreCase compares two file paths case-insensitively (Windows is case-preserving
// but case-insensitive for path comparisons).
func equalsIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// normalizeAutostartPath removes the optional outer quotes required by a Windows
// Run value before comparison. SetAutostart deliberately writes quoted paths so a
// portable executable remains valid after being moved into a directory with spaces.
func normalizeAutostartPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, `"`)
	return filepath.Clean(path)
}

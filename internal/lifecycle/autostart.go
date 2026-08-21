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

const autostartKeyPath = `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`
const autostartValueName = "Remotune"

type AutostartStatus struct {
	Registered     bool
	PathMatch      bool
	RegisteredPath string
}

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
func normalizeAutostartPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, `"`)
	return filepath.Clean(path)
}

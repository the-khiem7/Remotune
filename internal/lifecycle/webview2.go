//go:build windows

package lifecycle

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// webView2ClientGUID is the EdgeUpdate client GUID that stores WebView2's installed
// version under its "pv" value. Checking this key across HKLM 64-bit, HKLM
// WOW6432Node, and HKCU covers per-machine and per-user installs (Phase 0 evidence).
const webView2ClientGUID = `{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`

// CheckWebView2 returns nil if the WebView2 runtime is detected, or an error
// explaining what is missing. It reads three registry locations following the Phase 0
// evidence strategy, covering system-wide 64-bit, system-wide 32-bit (WOW6432Node),
// and per-user installs.
func CheckWebView2() error {
	paths := []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\EdgeUpdate\Clients\` + webView2ClientGUID},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\` + webView2ClientGUID},
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\EdgeUpdate\Clients\` + webView2ClientGUID},
	}

	for _, p := range paths {
		k, err := registry.OpenKey(p.root, p.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		defer k.Close()

		pv, _, err := k.GetStringValue("pv")
		if err != nil {
			continue
		}
		if pv != "" && pv != "0.0.0.0" {
			return nil // WebView2 is installed.
		}
	}

	return errors.New("WebView2 runtime not found; install it from https://developer.microsoft.com/en-us/microsoft-edge/webview2/")
}

// WebView2Version returns the installed WebView2 version string, or an error.
func WebView2Version() (string, error) {
	paths := []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\EdgeUpdate\Clients\` + webView2ClientGUID},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\` + webView2ClientGUID},
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\EdgeUpdate\Clients\` + webView2ClientGUID},
	}

	for _, p := range paths {
		k, err := registry.OpenKey(p.root, p.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		defer k.Close()

		pv, _, err := k.GetStringValue("pv")
		if err != nil {
			continue
		}
		if pv != "" && pv != "0.0.0.0" {
			return pv, nil
		}
	}

	return "", fmt.Errorf("WebView2 not detected")
}

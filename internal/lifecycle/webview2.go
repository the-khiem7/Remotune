//go:build windows

package lifecycle

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const webView2ClientGUID = `{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`

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

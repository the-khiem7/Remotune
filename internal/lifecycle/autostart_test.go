//go:build windows

package lifecycle

import "testing"

func TestNormalizeAutostartPathAcceptsQuotedRunValue(t *testing.T) {
	plain := `D:\Remotune Builds\remotune-v0.1.5.exe`
	quoted := `"D:\Remotune Builds\remotune-v0.1.5.exe"`
	if got := normalizeAutostartPath(quoted); !equalsIgnoreCase(got, normalizeAutostartPath(plain)) {
		t.Fatalf("quoted Run value normalized to %q, want %q", got, plain)
	}
}

func TestNormalizeAutostartPathTrimsWhitespace(t *testing.T) {
	if got, want := normalizeAutostartPath(`  "D:\Remotune\remotune.exe"  `), `D:\Remotune\remotune.exe`; got != want {
		t.Fatalf("normalized path = %q, want %q", got, want)
	}
}

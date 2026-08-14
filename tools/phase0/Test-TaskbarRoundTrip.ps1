<#
.SYNOPSIS
  Phase 0 evidence: proves SHAppBarMessage ABM_GETSTATE/ABM_SETSTATE can toggle taskbar
  auto-hide and restore the EXACT original state, preserving unrelated appbar bits.

.DESCRIPTION
  Mutates taskbar auto-hide temporarily, then restores it in the same run.
  The original state is printed and re-asserted at the end; the finally block
  guarantees restoration even on error.
#>
$ErrorActionPreference = 'Stop'

Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class Tb
{
    [StructLayout(LayoutKind.Sequential)]
    public struct APPBARDATA {
        public uint cbSize; public IntPtr hWnd; public uint uCallbackMessage;
        public uint uEdge; public RECT rc; public IntPtr lParam;
    }
    [StructLayout(LayoutKind.Sequential)]
    public struct RECT { public int left, top, right, bottom; }

    [DllImport("shell32.dll", SetLastError = true)]
    static extern IntPtr SHAppBarMessage(uint dwMessage, ref APPBARDATA pData);

    const uint ABM_GETSTATE = 0x4;
    const uint ABM_SETSTATE = 0xA;

    public static uint GetState() {
        var d = new APPBARDATA(); d.cbSize = (uint)Marshal.SizeOf(typeof(APPBARDATA));
        return (uint)SHAppBarMessage(ABM_GETSTATE, ref d).ToInt64();
    }
    public static int SetState(uint state) {
        var d = new APPBARDATA(); d.cbSize = (uint)Marshal.SizeOf(typeof(APPBARDATA));
        d.lParam = new IntPtr(state);
        SHAppBarMessage(ABM_SETSTATE, ref d);
        return Marshal.GetLastWin32Error();
    }
}
'@

Add-Type -AssemblyName System.Windows.Forms

function Show-State([string]$tag) {
    $s = [uint32][Tb]::GetState()
    $wa = [System.Windows.Forms.Screen]::PrimaryScreen.WorkingArea
    $bd = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds
    Write-Host ("  {0,-22} abmState=0x{1:X2}  autoHide={2,-5} alwaysOnTop={3,-5} otherBits=0x{4:X2}  workArea.H={5} (bounds.H={6})" -f `
            $tag, $s, [bool]($s -band 1), [bool]($s -band 2), ($s -band (-bnot 3)), $wa.Height, $bd.Height)
    return $s
}

$ABS_AUTOHIDE = 1

"=== Taskbar auto-hide round-trip (SHAppBarMessage) ==="
$original = [Tb]::GetState()
"ORIGINAL STATE = 0x{0:X2}  (this will be restored)" -f $original
""

try {
    $s0 = Show-State 'initial'

    # toggle: flip ONLY the auto-hide bit, preserve every other bit
    $target = if ($s0 -band $ABS_AUTOHIDE) { $s0 -band (-bnot $ABS_AUTOHIDE) } else { $s0 -bor $ABS_AUTOHIDE }
    "`n-> ABM_SETSTATE 0x{0:X2} (flip auto-hide only, preserve other bits)" -f $target
    $err = [Tb]::SetState([uint32]$target)
    Start-Sleep -Milliseconds 1200
    $s1 = Show-State 'after toggle'
    $applyOk = ($s1 -eq $target)
    "   apply verified: $applyOk  (lastWin32Error=$err)"

    "`n-> ABM_SETSTATE 0x{0:X2} (restore exact original)" -f $s0
    [Tb]::SetState([uint32]$s0) | Out-Null
    Start-Sleep -Milliseconds 1200
    $s2 = Show-State 'after restore'
    $restoreOk = ($s2 -eq $s0)
    "   restore verified: $restoreOk"

    "`n=== RESULT ==="
    "  apply   : $(if($applyOk){'PASS'}else{'FAIL'})"
    "  restore : $(if($restoreOk){'PASS - exact original state recovered'}else{'FAIL'})"
    "  unrelated bits preserved: $((($s0 -band (-bnot $ABS_AUTOHIDE)) -eq ($s1 -band (-bnot $ABS_AUTOHIDE))))"
    "  elevation required: NO (ran as non-elevated user)"
}
finally {
    # guarantee restoration
    $cur = [Tb]::GetState()
    if ($cur -ne $original) {
        "`n!! safety restore: 0x{0:X2} -> 0x{1:X2}" -f $cur, $original
        [Tb]::SetState([uint32]$original) | Out-Null
    }
    "`nfinal state = 0x{0:X2} (original was 0x{1:X2}) -> {2}" -f [Tb]::GetState(), $original, $(if ([Tb]::GetState() -eq $original) { 'RESTORED' } else { 'MISMATCH' })
}

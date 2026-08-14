<#
.SYNOPSIS
  Phase 0/1 evidence: does a WM_SETTINGCHANGE broadcast revert a taskbar auto-hide
  override that was applied through ABM_SETSTATE?

.DESCRIPTION
  Hypothesis under test:
    ABM_SETSTATE changes the LIVE appbar state but does not reliably persist to
    StuckRights3. A settings-change broadcast makes Explorer reload the persisted
    value, silently undoing the override.

  If true, Remotune's own Visual Effects apply (which broadcasts) can revert its own
  taskbar override, and ordering plus post-broadcast verification become mandatory.
#>
$ErrorActionPreference = 'Stop'

Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class Tb2
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

    [DllImport("user32.dll", CharSet = CharSet.Auto)]
    static extern IntPtr SendMessageTimeout(IntPtr h, uint m, IntPtr w, string l, uint f, uint t, out UIntPtr r);

    public static uint Get() {
        var d = new APPBARDATA(); d.cbSize = (uint)Marshal.SizeOf(typeof(APPBARDATA));
        return (uint)SHAppBarMessage(0x4, ref d).ToInt64();
    }
    public static void Set(uint s) {
        var d = new APPBARDATA(); d.cbSize = (uint)Marshal.SizeOf(typeof(APPBARDATA));
        d.lParam = new IntPtr(s);
        SHAppBarMessage(0xA, ref d);
    }
    public static void Broadcast(string area) {
        UIntPtr r;
        SendMessageTimeout(new IntPtr(0xFFFF), 0x001A, IntPtr.Zero, area, 0x0002, 1000, out r);
    }
}
'@ -ErrorAction SilentlyContinue

function SR8 { (Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\StuckRects3').Settings[8] }
function Show($tag) {
    $a = [Tb2]::Get(); $s = SR8
    Write-Host ("  {0,-30} ABM=0x{1:X2} (autoHide={2,-5})   StuckRects3[8]=0x{3:X2} (autoHide={4,-5})   {5}" -f `
            $tag, $a, [bool]($a -band 1), $s, [bool]($s -band 1), $(if ([bool]($a -band 1) -eq [bool]($s -band 1)) { 'agree' } else { '*** DIVERGED ***' }))
}

Write-Host '=== taskbar override vs WM_SETTINGCHANGE broadcast ==='
Write-Host ''
Show 'initial'

Write-Host "`n-> ABM_SETSTATE auto-hide ON (0x01)"
[Tb2]::Set(1); Start-Sleep -Milliseconds 1200
Show 'after ABM_SETSTATE ON'

Write-Host "`n-> broadcast WM_SETTINGCHANGE 'WindowMetrics'"
[Tb2]::Broadcast('WindowMetrics'); Start-Sleep -Milliseconds 1500
$afterBroadcast = [Tb2]::Get()
Show 'after broadcast'

Write-Host "`n=== RESULT ==="
if (($afterBroadcast -band 1) -eq 0) {
    Write-Host '  CONFIRMED: the broadcast reverted the auto-hide override.'
    Write-Host '  => ABM_SETSTATE alone is NOT durable across a settings-change broadcast.'
}
else {
    Write-Host '  NOT reproduced: the override survived the broadcast.'
    Write-Host '  => the earlier revert had another cause; investigate further.'
}

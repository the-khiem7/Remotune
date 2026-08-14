<#
.SYNOPSIS
  Phase 0/1 reference: applies Windows "Adjust for best performance" as a TRANSFORMATION.

.DESCRIPTION
  Implements ledger decision 53. This deliberately does NOT replay a captured
  Best Performance snapshot, because such a snapshot carries machine-specific values
  (FontSmoothingType, build-specific mask bits) that must not be pushed onto other machines.

  Instead it changes exactly the values the real Windows preset changes, and leaves
  everything else alone:

    - 11 effects forced OFF via SystemParametersInfo
    - 7 probed effects deliberately UNTOUCHED
    - discrete registry values set to their preset values (note IconsOnly goes 0 -> 1)
    - UserPreferencesMask: only the preset's bits are CLEARED, all other bits preserved

  This is the reference behaviour for VisualEffectsManager.ApplyBestPerformance().
#>
param([switch]$Quiet)

$ErrorActionPreference = 'Stop'

Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class BpSet
{
    [DllImport("user32.dll", SetLastError = true)]
    static extern bool SystemParametersInfo(uint a, uint p, IntPtr v, uint f);
    [StructLayout(LayoutKind.Sequential)]
    public struct ANIMATIONINFO { public uint cbSize; public int iMinAnimate; }
    [DllImport("user32.dll", SetLastError = true)]
    static extern bool SystemParametersInfo(uint a, uint p, ref ANIMATIONINFO v, uint f);
    [DllImport("user32.dll", CharSet = CharSet.Auto)]
    static extern IntPtr SendMessageTimeout(IntPtr h, uint m, IntPtr w, string l, uint f, uint t, out UIntPtr r);

    const uint FLAGS = 0x03; // SPIF_UPDATEINIFILE | SPIF_SENDCHANGE

    public static string ByUiParam(uint action, int value) {
        return SystemParametersInfo(action, (uint)value, IntPtr.Zero, FLAGS) ? "ok" : "ERR " + Marshal.GetLastWin32Error();
    }
    public static string ByPvParam(uint action, int value) {
        return SystemParametersInfo(action, 0, new IntPtr(value), FLAGS) ? "ok" : "ERR " + Marshal.GetLastWin32Error();
    }
    public static string MinAnimate(int value) {
        var ai = new ANIMATIONINFO();
        ai.cbSize = (uint)Marshal.SizeOf(typeof(ANIMATIONINFO));
        ai.iMinAnimate = value;
        return SystemParametersInfo(0x0049, ai.cbSize, ref ai, FLAGS) ? "ok" : "ERR " + Marshal.GetLastWin32Error();
    }
    public static void Broadcast() {
        UIntPtr r;
        SendMessageTimeout(new IntPtr(0xFFFF), 0x001A, IntPtr.Zero, "WindowMetrics", 0x0002, 1000, out r);
    }
}
'@ -ErrorAction SilentlyContinue

function Say($m) { if (-not $Quiet) { Write-Host $m } }

# --- effects the preset turns OFF: name -> @(SET action, style) ---
$Off = [ordered]@{
    'DragFullWindows'        = @(0x0025, 'ui')
    'FontSmoothing'          = @(0x004B, 'ui')
    'MenuAnimation'          = @(0x1003, 'pv')
    'ComboBoxAnimation'      = @(0x1005, 'pv')
    'ListBoxSmoothScrolling' = @(0x1007, 'pv')
    'SelectionFade'          = @(0x1015, 'pv')
    'TooltipAnimation'       = @(0x1017, 'pv')
    'CursorShadow'           = @(0x101B, 'pv')
    'DropShadow'             = @(0x1025, 'pv')
    'ClientAreaAnimation'    = @(0x1043, 'pv')
}

# --- effects the preset deliberately leaves alone (documented, not code) ---
$Untouched = @('GradientCaptions', 'KeyboardCues', 'HotTracking', 'MenuFade', 'TooltipFade', 'FlatMenu', 'UIEffects')

# --- discrete registry values the preset writes ---
$Reg = @(
    @('HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects', 'VisualFXSetting', 2),
    @('HKCU:\Control Panel\Desktop', 'DragFullWindows', '0'),
    @('HKCU:\Control Panel\Desktop', 'FontSmoothing', '0'),
    @('HKCU:\Control Panel\Desktop\WindowMetrics', 'MinAnimate', '0'),
    @('HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced', 'ListviewAlphaSelect', 0),
    @('HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced', 'ListviewShadow', 0),
    @('HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced', 'TaskbarAnimations', 0),
    @('HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced', 'IconsOnly', 1),
    @('HKCU:\Software\Microsoft\Windows\DWM', 'EnableAeroPeek', 0),
    @('HKCU:\Software\Microsoft\Windows\DWM', 'AlwaysHibernateThumbnails', 0)
)

# --- mask bits the preset CLEARS, per byte. every other bit is preserved. ---
$ClearMask = [byte[]](0x0E, 0x2C, 0x04, 0x00, 0x02, 0x00, 0x00, 0x00)

Say '=== ApplyBestPerformance (transformation) ==='

Say "`n--- step 1: SPI effects -> 0 ---"
foreach ($n in $Off.Keys) {
    $a = [uint32]$Off[$n][0]
    $r = if ($Off[$n][1] -eq 'ui') { [BpSet]::ByUiParam($a, 0) } else { [BpSet]::ByPvParam($a, 0) }
    Say ("  {0,-24} -> 0  [{1}]" -f $n, $r)
}
Say ("  {0,-24} -> 0  [{1}]" -f 'MinAnimate', [BpSet]::MinAnimate(0))
Say ("  untouched by design: {0}" -f ($Untouched -join ', '))

Say "`n--- step 2: discrete registry values ---"
foreach ($e in $Reg) {
    $p = $e[0]; $n = $e[1]; $v = $e[2]
    Set-ItemProperty -LiteralPath $p -Name $n -Value $v
    Say ("  {0,-28} -> {1}" -f $n, $v)
}

Say "`n--- step 3: clear preset mask bits, preserve the rest ---"
$cur = (Get-ItemProperty 'HKCU:\Control Panel\Desktop').UserPreferencesMask
$new = [byte[]]::new($cur.Length)
for ($i = 0; $i -lt $cur.Length; $i++) {
    $cm = if ($i -lt $ClearMask.Length) { $ClearMask[$i] } else { 0 }
    $new[$i] = $cur[$i] -band (-bnot $cm)
}
Set-ItemProperty -LiteralPath 'HKCU:\Control Panel\Desktop' -Name 'UserPreferencesMask' -Value $new
Say ("  {0}  ->  {1}" -f (($cur | ForEach-Object { $_.ToString('X2') }) -join ' '), (($new | ForEach-Object { $_.ToString('X2') }) -join ' '))

Say "`n--- step 4: broadcast ---"
[BpSet]::Broadcast()
Start-Sleep -Milliseconds 800
Say '  done'

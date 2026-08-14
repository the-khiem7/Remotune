<#
.SYNOPSIS
  Phase 0 evidence: proves an exact programmatic restore of a captured Visual Effects
  snapshot, including bits that map to no documented effect.

.DESCRIPTION
  This is the reference behaviour for VisualEffectsManager.Restore(snapshot).

  Order matters:
    1. per-effect SystemParametersInfo writes  (updates the live session)
    2. discrete registry values                (Explorer / DWM backed)
    3. UserPreferencesMask written verbatim    (restores UNATTRIBUTED bits last,
                                                so no SPI write can clobber them)
    4. WM_SETTINGCHANGE broadcast

.EXAMPLE
  .\Restore-VisualState.ps1 -Label 01-baseline-bestappearance
#>
param(
    [Parameter(Mandatory = $true)][string]$Label,
    [string]$OutDir = (Join-Path $PSScriptRoot 'snapshots'),
    [switch]$WhatIfOnly
)

$ErrorActionPreference = 'Stop'

Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;

public static class SpiSet
{
    // value carried in uiParam
    [DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
    static extern bool SystemParametersInfo(uint uiAction, uint uiParam, IntPtr pvParam, uint fWinIni);

    [StructLayout(LayoutKind.Sequential)]
    public struct ANIMATIONINFO { public uint cbSize; public int iMinAnimate; }

    [DllImport("user32.dll", SetLastError = true)]
    static extern bool SystemParametersInfo(uint uiAction, uint uiParam, ref ANIMATIONINFO pvParam, uint fWinIni);

    [DllImport("user32.dll", CharSet = CharSet.Auto)]
    static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, IntPtr wParam, string lParam,
                                            uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);

    const uint SPIF_UPDATEINIFILE = 0x01;
    const uint SPIF_SENDCHANGE = 0x02;
    const uint FLAGS = SPIF_UPDATEINIFILE | SPIF_SENDCHANGE;

    // effects whose value travels in uiParam
    public static string SetByUiParam(uint action, int value) {
        bool ok = SystemParametersInfo(action, (uint)value, IntPtr.Zero, FLAGS);
        return ok ? "ok" : "ERR " + Marshal.GetLastWin32Error();
    }

    // effects whose value travels in pvParam as a BOOL
    public static string SetByPvParam(uint action, int value) {
        bool ok = SystemParametersInfo(action, 0, new IntPtr(value), FLAGS);
        return ok ? "ok" : "ERR " + Marshal.GetLastWin32Error();
    }

    public static string SetMinAnimate(int value) {
        var ai = new ANIMATIONINFO();
        ai.cbSize = (uint)Marshal.SizeOf(typeof(ANIMATIONINFO));
        ai.iMinAnimate = value;
        bool ok = SystemParametersInfo(0x0049, ai.cbSize, ref ai, FLAGS);
        return ok ? "ok" : "ERR " + Marshal.GetLastWin32Error();
    }

    public static void Broadcast() {
        UIntPtr r;
        SendMessageTimeout(new IntPtr(0xFFFF), 0x001A, IntPtr.Zero, "WindowMetrics", 0x0002, 1000, out r);
        SendMessageTimeout(new IntPtr(0xFFFF), 0x001A, IntPtr.Zero, "Environment", 0x0002, 1000, out r);
    }
}
'@

# effect name -> @(SET action, style)  style: 'ui' = value in uiParam, 'pv' = value in pvParam
$SetMap = [ordered]@{
    'DragFullWindows'        = @(0x0025, 'ui')
    'FontSmoothing'          = @(0x004B, 'ui')
    'MenuAnimation'          = @(0x1003, 'pv')
    'ComboBoxAnimation'      = @(0x1005, 'pv')
    'ListBoxSmoothScrolling' = @(0x1007, 'pv')
    'GradientCaptions'       = @(0x1009, 'pv')
    'KeyboardCues'           = @(0x100B, 'pv')
    'HotTracking'            = @(0x100F, 'pv')
    'MenuFade'               = @(0x1013, 'pv')
    'SelectionFade'          = @(0x1015, 'pv')
    'TooltipAnimation'       = @(0x1017, 'pv')
    'TooltipFade'            = @(0x1019, 'pv')
    'CursorShadow'           = @(0x101B, 'pv')
    'FlatMenu'               = @(0x1023, 'pv')
    'DropShadow'             = @(0x1025, 'pv')
    'UIEffects'              = @(0x103F, 'pv')
    'ClientAreaAnimation'    = @(0x1043, 'pv')
}

# snapshot registry key -> real registry path + value name
$RegMap = [ordered]@{
    'VisualFXSetting'                 = @('HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects', 'VisualFXSetting')
    'Desktop.DragFullWindows'         = @('HKCU:\Control Panel\Desktop', 'DragFullWindows')
    'Desktop.FontSmoothing'           = @('HKCU:\Control Panel\Desktop', 'FontSmoothing')
    'Desktop.FontSmoothingType'       = @('HKCU:\Control Panel\Desktop', 'FontSmoothingType')
    'Desktop.MenuShowDelay'           = @('HKCU:\Control Panel\Desktop', 'MenuShowDelay')
    'WindowMetrics.MinAnimate'        = @('HKCU:\Control Panel\Desktop\WindowMetrics', 'MinAnimate')
    'Advanced.ListviewAlphaSelect'    = @('HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced', 'ListviewAlphaSelect')
    'Advanced.ListviewShadow'         = @('HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced', 'ListviewShadow')
    'Advanced.TaskbarAnimations'      = @('HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced', 'TaskbarAnimations')
    'Advanced.IconsOnly'              = @('HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced', 'IconsOnly')
    'DWM.EnableAeroPeek'              = @('HKCU:\Software\Microsoft\Windows\DWM', 'EnableAeroPeek')
    'DWM.AlwaysHibernateThumbnails'   = @('HKCU:\Software\Microsoft\Windows\DWM', 'AlwaysHibernateThumbnails')
    'DWM.Composition'                 = @('HKCU:\Software\Microsoft\Windows\DWM', 'Composition')
}

$path = Join-Path $OutDir "$Label.json"
if (-not (Test-Path $path)) { throw "snapshot not found: $path" }
$snap = Get-Content $path -Raw | ConvertFrom-Json

"=== restoring snapshot '$Label' ==="
"target VisualFXSetting     = $($snap.registry.VisualFXSetting)"
"target UserPreferencesMask = $($snap.registry.UserPreferencesMask)"
if ($WhatIfOnly) { "WhatIfOnly set - no changes made"; return }

"`n--- step 1: SystemParametersInfo per-effect writes ---"
foreach ($name in $SetMap.Keys) {
    if ($null -eq $snap.spi.$name) { "  {0,-24} (absent in snapshot, skipped)" -f $name; continue }
    $val = [int]$snap.spi.$name
    $action = [uint32]$SetMap[$name][0]
    $style = $SetMap[$name][1]
    $r = if ($style -eq 'ui') { [SpiSet]::SetByUiParam($action, $val) } else { [SpiSet]::SetByPvParam($action, $val) }
    "  {0,-24} = {1}  [{2}]" -f $name, $val, $r
}
if ($null -ne $snap.spi.MinAnimate) {
    "  {0,-24} = {1}  [{2}]" -f 'MinAnimate', [int]$snap.spi.MinAnimate, [SpiSet]::SetMinAnimate([int]$snap.spi.MinAnimate)
}

"`n--- step 2: discrete registry values ---"
foreach ($k in $RegMap.Keys) {
    $want = $snap.registry.$k
    if ($null -eq $want) { continue }
    $rp = $RegMap[$k][0]; $rn = $RegMap[$k][1]
    if (-not (Test-Path $rp)) { "  {0,-34} (key missing, skipped)" -f $k; continue }
    try {
        $item = Get-Item -LiteralPath $rp
        $kind = if ($item.GetValueNames() -contains $rn) { $item.GetValueKind($rn) } else { 'DWord' }
        $typed = if ($kind -eq 'String') { [string]$want } else { [int]$want }
        Set-ItemProperty -LiteralPath $rp -Name $rn -Value $typed
        "  {0,-34} = {1,-6} [{2}]" -f $k, $want, $kind
    }
    catch { "  {0,-34} FAILED: {1}" -f $k, $_.Exception.Message }
}

"`n--- step 3: UserPreferencesMask verbatim (restores unattributed bits) ---"
$hex = ([string]$snap.registry.UserPreferencesMask) -replace '^0x', ''
$bytes = [byte[]](0..(($hex.Length / 2) - 1) | ForEach-Object { [Convert]::ToByte($hex.Substring($_ * 2, 2), 16) })
Set-ItemProperty -LiteralPath 'HKCU:\Control Panel\Desktop' -Name 'UserPreferencesMask' -Value $bytes
"  wrote $($bytes.Length) bytes: $(($bytes | ForEach-Object { $_.ToString('X2') }) -join ' ')"

"`n--- step 4: broadcast WM_SETTINGCHANGE ---"
[SpiSet]::Broadcast()
"  broadcast sent"
Start-Sleep -Milliseconds 800
"`nrestore pass complete"

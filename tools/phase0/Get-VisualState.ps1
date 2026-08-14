<#
.SYNOPSIS
  Phase 0 evidence tool. Captures a complete snapshot of every Windows value that
  the Performance Options "Visual Effects" surface and taskbar auto-hide can affect,
  then lets two snapshots be diffed.

.DESCRIPTION
  Read-only. Never mutates Windows state.

  Capture:  .\Get-VisualState.ps1 -Label before
  Diff:     .\Get-VisualState.ps1 -Diff before,after

.NOTES
  Snapshot layout is the candidate schema for VisualEffectsManager.Snapshot().
#>
[CmdletBinding(DefaultParameterSetName = 'Capture')]
param(
    [Parameter(ParameterSetName = 'Capture', Mandatory = $true)]
    [string]$Label,

    [Parameter(ParameterSetName = 'Diff', Mandatory = $true)]
    [string[]]$Diff,

    [string]$OutDir = (Join-Path $PSScriptRoot 'snapshots')
)

$ErrorActionPreference = 'Stop'

Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;

public static class P0Native
{
    [DllImport("user32.dll", SetLastError = true)]
    static extern bool SystemParametersInfo(uint a, uint p, ref int v, uint f);

    [DllImport("user32.dll", SetLastError = true)]
    static extern bool SystemParametersInfo(uint a, uint p, ref ANIMATIONINFO v, uint f);

    [StructLayout(LayoutKind.Sequential)]
    public struct ANIMATIONINFO { public uint cbSize; public int iMinAnimate; }

    [StructLayout(LayoutKind.Sequential)]
    public struct APPBARDATA {
        public uint cbSize; public IntPtr hWnd; public uint uCallbackMessage;
        public uint uEdge; public RECT rc; public IntPtr lParam;
    }
    [StructLayout(LayoutKind.Sequential)]
    public struct RECT { public int left, top, right, bottom; }

    [DllImport("shell32.dll", SetLastError = true)]
    static extern IntPtr SHAppBarMessage(uint dwMessage, ref APPBARDATA pData);

    [DllImport("dwmapi.dll")]
    static extern int DwmIsCompositionEnabled(out bool enabled);

    public static int Spi(uint action) {
        int v = -1;
        return SystemParametersInfo(action, 0, ref v, 0) ? v : -999;
    }

    public static int MinAnimate() {
        var ai = new ANIMATIONINFO();
        ai.cbSize = (uint)Marshal.SizeOf(typeof(ANIMATIONINFO));
        return SystemParametersInfo(0x0048, ai.cbSize, ref ai, 0) ? ai.iMinAnimate : -999;
    }

    public static uint AppBarState() {
        var d = new APPBARDATA();
        d.cbSize = (uint)Marshal.SizeOf(typeof(APPBARDATA));
        return (uint)SHAppBarMessage(0x00000004, ref d).ToInt64(); // ABM_GETSTATE
    }

    public static string TaskbarRect() {
        var d = new APPBARDATA();
        d.cbSize = (uint)Marshal.SizeOf(typeof(APPBARDATA));
        if (SHAppBarMessage(0x00000005, ref d) == IntPtr.Zero) return "FAILED"; // ABM_GETTASKBARPOS
        return d.uEdge + ":" + d.rc.left + "," + d.rc.top + "," + d.rc.right + "," + d.rc.bottom;
    }

    public static bool Dwm() { bool e; return DwmIsCompositionEnabled(out e) == 0 && e; }
}
'@ -ErrorAction SilentlyContinue

# SPI_GET* actions that back the Performance Options checkboxes
$SpiMap = [ordered]@{
    'DragFullWindows'        = 0x0026
    'FontSmoothing'          = 0x004A
    'MenuAnimation'          = 0x1002
    'ComboBoxAnimation'      = 0x1004
    'ListBoxSmoothScrolling' = 0x1006
    'GradientCaptions'       = 0x1008
    'KeyboardCues'           = 0x100A
    'HotTracking'            = 0x100E
    'MenuFade'               = 0x1012
    'SelectionFade'          = 0x1014
    'TooltipAnimation'       = 0x1016
    'TooltipFade'            = 0x1018
    'CursorShadow'           = 0x101A
    'FlatMenu'               = 0x1022
    'DropShadow'             = 0x1024
    'UIEffects'              = 0x103E
    'ClientAreaAnimation'    = 0x1042
}

function Get-RegVal($path, $name) {
    try {
        $k = Get-Item -LiteralPath $path -ErrorAction Stop
        if ($k.GetValueNames() -notcontains $name) { return $null }
        $v = $k.GetValue($name)
        if ($v -is [byte[]]) { return '0x' + (($v | ForEach-Object { $_.ToString('X2') }) -join '') }
        return $v
    }
    catch { return $null }
}

function New-Snapshot {
    $spi = [ordered]@{}
    foreach ($k in $SpiMap.Keys) { $spi[$k] = [P0Native]::Spi([uint32]$SpiMap[$k]) }
    $spi['MinAnimate'] = [P0Native]::MinAnimate()

    $reg = [ordered]@{}
    $reg['VisualFXSetting'] = Get-RegVal 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects' 'VisualFXSetting'
    $reg['UserPreferencesMask'] = Get-RegVal 'HKCU:\Control Panel\Desktop' 'UserPreferencesMask'
    $reg['Desktop.DragFullWindows'] = Get-RegVal 'HKCU:\Control Panel\Desktop' 'DragFullWindows'
    $reg['Desktop.FontSmoothing'] = Get-RegVal 'HKCU:\Control Panel\Desktop' 'FontSmoothing'
    $reg['Desktop.FontSmoothingType'] = Get-RegVal 'HKCU:\Control Panel\Desktop' 'FontSmoothingType'
    $reg['Desktop.MenuShowDelay'] = Get-RegVal 'HKCU:\Control Panel\Desktop' 'MenuShowDelay'
    $reg['WindowMetrics.MinAnimate'] = Get-RegVal 'HKCU:\Control Panel\Desktop\WindowMetrics' 'MinAnimate'
    foreach ($n in 'ListviewAlphaSelect', 'ListviewShadow', 'TaskbarAnimations', 'IconsOnly', 'ShowTaskViewButton') {
        $reg["Advanced.$n"] = Get-RegVal 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced' $n
    }
    foreach ($n in 'EnableAeroPeek', 'AlwaysHibernateThumbnails', 'Composition') {
        $reg["DWM.$n"] = Get-RegVal 'HKCU:\Software\Microsoft\Windows\DWM' $n
    }

    $vfxSub = [ordered]@{}
    $vfxRoot = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects'
    if (Test-Path $vfxRoot) {
        Get-ChildItem $vfxRoot | Sort-Object PSChildName | ForEach-Object {
            $vfxSub[$_.PSChildName] = Get-RegVal $_.PSPath 'DefaultApplied'
        }
    }

    Add-Type -AssemblyName System.Windows.Forms -ErrorAction SilentlyContinue
    $screens = @([System.Windows.Forms.Screen]::AllScreens | ForEach-Object {
            [ordered]@{ device = $_.DeviceName; primary = $_.Primary; bounds = $_.Bounds.ToString(); workArea = $_.WorkingArea.ToString() }
        })

    $abs = [P0Native]::AppBarState()
    return [ordered]@{
        schemaVersion = 1
        capturedAt    = (Get-Date).ToString('o')
        machine       = $env:COMPUTERNAME
        osBuild       = "$([System.Environment]::OSVersion.Version) UBR=$((Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion').UBR)"
        spi           = $spi
        registry      = $reg
        visualFxSub   = $vfxSub
        dwmComposition = [P0Native]::Dwm()
        taskbar       = [ordered]@{
            abmState      = $abs
            absAutoHide   = [bool]($abs -band 1)
            absAlwaysOnTop = [bool]($abs -band 2)
            otherBits     = ($abs -band (-bnot 3))
            taskbarRect   = [P0Native]::TaskbarRect()
        }
        screens       = $screens
    }
}

function Flatten($obj, $prefix = '') {
    $out = [ordered]@{}
    foreach ($k in $obj.Keys) {
        $v = $obj[$k]
        $key = if ($prefix) { "$prefix.$k" } else { $k }
        if ($v -is [System.Collections.Specialized.OrderedDictionary] -or $v -is [hashtable]) {
            foreach ($e in (Flatten $v $key).GetEnumerator()) { $out[$e.Key] = $e.Value }
        }
        else { $out[$key] = $v }
    }
    return $out
}

if (-not (Test-Path $OutDir)) { New-Item -ItemType Directory -Path $OutDir -Force | Out-Null }

if ($PSCmdlet.ParameterSetName -eq 'Capture') {
    $snap = New-Snapshot
    $path = Join-Path $OutDir "$Label.json"
    $snap | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $path -Encoding utf8
    "captured '$Label' -> $path"
    "  VisualFXSetting     = $($snap.registry.VisualFXSetting)"
    "  UserPreferencesMask = $($snap.registry.UserPreferencesMask)"
    "  taskbar autoHide    = $($snap.taskbar.absAutoHide)  (abmState=0x$('{0:X2}' -f $snap.taskbar.abmState))"
    "  workArea            = $($snap.screens[0].workArea)"
}
else {
    if ($Diff.Count -ne 2) { throw 'need exactly two labels: -Diff before,after' }
    $a = Get-Content (Join-Path $OutDir "$($Diff[0]).json") -Raw | ConvertFrom-Json
    $b = Get-Content (Join-Path $OutDir "$($Diff[1]).json") -Raw | ConvertFrom-Json

    function ToOrdered($o) {
        $d = [ordered]@{}
        foreach ($p in $o.PSObject.Properties) {
            if ($p.Value -is [System.Management.Automation.PSCustomObject]) { $d[$p.Name] = ToOrdered $p.Value }
            elseif ($p.Value -is [Array]) { $d[$p.Name] = ($p.Value | ForEach-Object { if ($_ -is [System.Management.Automation.PSCustomObject]) { (ToOrdered $_ | ConvertTo-Json -Compress) } else { $_ } }) -join ' | ' }
            else { $d[$p.Name] = $p.Value }
        }
        return $d
    }

    $fa = Flatten (ToOrdered $a)
    $fb = Flatten (ToOrdered $b)
    $keys = ($fa.Keys + $fb.Keys) | Select-Object -Unique
    $changed = @()
    foreach ($k in $keys) {
        if ($k -in 'capturedAt') { continue }
        $va = if ($fa.Contains($k)) { $fa[$k] } else { '<absent>' }
        $vb = if ($fb.Contains($k)) { $fb[$k] } else { '<absent>' }
        if ("$va" -ne "$vb") { $changed += [pscustomobject]@{ Key = $k; Before = "$va"; After = "$vb" } }
    }
    "=== DIFF $($Diff[0]) -> $($Diff[1]) ==="
    if ($changed.Count -eq 0) { "  NO DIFFERENCES (states are identical)" }
    else {
        $changed | Format-Table -AutoSize -Wrap Key, Before, After
        "  changed values: $($changed.Count)"
    }
}

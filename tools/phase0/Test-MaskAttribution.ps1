<#
.SYNOPSIS
  Attributes every UserPreferencesMask bit to a concrete SPI effect by toggling each
  effect and observing which bit moves.

.DESCRIPTION
  Phase 0 cross-checked only 13 effects and therefore reported some set bits as
  "unattributed". This test covers all probed effects, so the remaining genuinely
  unknown bits can be stated precisely.

  Each effect is restored to its original value immediately after being toggled.
#>
$ErrorActionPreference = 'Stop'

Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class Ma
{
    [DllImport("user32.dll", SetLastError=true)] static extern bool SystemParametersInfo(uint a, uint p, ref int v, uint f);
    [DllImport("user32.dll", SetLastError=true)] static extern bool SystemParametersInfo(uint a, uint p, IntPtr v, uint f);
    public static int Get(uint a) { int v=-1; return SystemParametersInfo(a,0,ref v,0) ? v : -999; }
    public static void SetUi(uint a, int v) { SystemParametersInfo(a,(uint)v,IntPtr.Zero,0x01); }
    public static void SetPv(uint a, int v) { SystemParametersInfo(a,0,new IntPtr(v),0x01); }
}
'@ -ErrorAction SilentlyContinue

# name -> @(GET, SET, style)
$E = [ordered]@{
    'DragFullWindows'        = @(0x0026, 0x0025, 'ui')
    'FontSmoothing'          = @(0x004A, 0x004B, 'ui')
    'MenuAnimation'          = @(0x1002, 0x1003, 'pv')
    'ComboBoxAnimation'      = @(0x1004, 0x1005, 'pv')
    'ListBoxSmoothScrolling' = @(0x1006, 0x1007, 'pv')
    'GradientCaptions'       = @(0x1008, 0x1009, 'pv')
    'KeyboardCues'           = @(0x100A, 0x100B, 'pv')
    'HotTracking'            = @(0x100E, 0x100F, 'pv')
    'MenuFade'               = @(0x1012, 0x1013, 'pv')
    'SelectionFade'          = @(0x1014, 0x1015, 'pv')
    'TooltipAnimation'       = @(0x1016, 0x1017, 'pv')
    'TooltipFade'            = @(0x1018, 0x1019, 'pv')
    'CursorShadow'           = @(0x101A, 0x101B, 'pv')
    'FlatMenu'               = @(0x1022, 0x1023, 'pv')
    'DropShadow'             = @(0x1024, 0x1025, 'pv')
    'UIEffects'              = @(0x103E, 0x103F, 'pv')
    'ClientAreaAnimation'    = @(0x1042, 0x1043, 'pv')
}

function Mask { (Get-ItemProperty 'HKCU:\Control Panel\Desktop').UserPreferencesMask }
function MaskStr($b) { ($b | ForEach-Object { $_.ToString('X2') }) -join ' ' }

function DeltaOf($a, $b) {
    $out = @()
    for ($i = 0; $i -lt [Math]::Min($a.Length, $b.Length); $i++) {
        $d = $a[$i] -bxor $b[$i]
        if ($d -ne 0) {
            foreach ($bit in 1, 2, 4, 8, 16, 32, 64, 128) {
                if ($d -band $bit) { $out += ("byte[{0}]:0x{1:X2}" -f $i, $bit) }
            }
        }
    }
    return $out
}

Write-Host "=== UserPreferencesMask bit attribution ===`n"
Write-Host "start mask: $(MaskStr (Mask))`n"
Write-Host ("{0,-24} {1,-8} {2}" -f 'Effect', 'Value', 'Mask bit(s) it controls')
Write-Host ('-' * 66)

$found = @{}
foreach ($n in $E.Keys) {
    $get = [uint32]$E[$n][0]; $set = [uint32]$E[$n][1]; $style = $E[$n][2]
    $orig = [Ma]::Get($get)
    if ($orig -lt 0) { Write-Host ("{0,-24} {1,-8} read failed" -f $n, '?'); continue }

    $before = Mask
    $flip = if ($orig -eq 0) { 1 } else { 0 }
    if ($style -eq 'ui') { [Ma]::SetUi($set, $flip) } else { [Ma]::SetPv($set, $flip) }
    Start-Sleep -Milliseconds 120
    $after = Mask
    $delta = DeltaOf $before $after

    # restore immediately
    if ($style -eq 'ui') { [Ma]::SetUi($set, $orig) } else { [Ma]::SetPv($set, $orig) }
    Start-Sleep -Milliseconds 120

    $desc = if ($delta.Count -eq 0) { '(no mask bit - stored elsewhere)' } else { $delta -join ', ' }
    foreach ($d in $delta) { $found[$d] = $n }
    Write-Host ("{0,-24} {1,-8} {2}" -f $n, $orig, $desc)
}

Write-Host "`nend mask:   $(MaskStr (Mask))"

Write-Host "`n=== attribution of every SET bit in the current mask ==="
$m = Mask
for ($i = 0; $i -lt $m.Length; $i++) {
    foreach ($bit in 1, 2, 4, 8, 16, 32, 64, 128) {
        if ($m[$i] -band $bit) {
            $k = "byte[$i]:0x$('{0:X2}' -f $bit)"
            $owner = if ($found.ContainsKey($k)) { $found[$k] } else { 'STILL UNKNOWN' }
            Write-Host ("  {0,-16} = {1}" -f $k, $owner)
        }
    }
}

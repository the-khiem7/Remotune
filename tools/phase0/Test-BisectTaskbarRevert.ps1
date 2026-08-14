<#
.SYNOPSIS
  Isolates which part of a Visual Effects apply reverts a live taskbar auto-hide override.

.DESCRIPTION
  Precondition: ABM live state = auto-hide ON while StuckRects3 persists OFF (diverged).
  Each phase of the apply runs in isolation and the live ABM state is re-read after it.
  The first phase that flips ABM back to OFF is the culprit.
#>
$ErrorActionPreference = 'Stop'

Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class Bi
{
    [StructLayout(LayoutKind.Sequential)]
    public struct A { public uint cbSize; public IntPtr hWnd; public uint m; public uint e; public int l,t,r,b; public IntPtr lp; }
    [DllImport("shell32.dll")] static extern IntPtr SHAppBarMessage(uint m, ref A d);
    [DllImport("user32.dll", SetLastError=true)] static extern bool SystemParametersInfo(uint a, uint p, IntPtr v, uint f);
    [StructLayout(LayoutKind.Sequential)] public struct ANIMATIONINFO { public uint cbSize; public int iMinAnimate; }
    [DllImport("user32.dll", SetLastError=true)] static extern bool SystemParametersInfo(uint a, uint p, ref ANIMATIONINFO v, uint f);
    [DllImport("user32.dll", CharSet=CharSet.Auto)] static extern IntPtr SendMessageTimeout(IntPtr h, uint m, IntPtr w, string l, uint f, uint t, out UIntPtr r);

    public static uint Get() { var d = new A(); d.cbSize=(uint)Marshal.SizeOf(typeof(A)); return (uint)SHAppBarMessage(4, ref d).ToInt64(); }
    public static void Set(uint s) { var d = new A(); d.cbSize=(uint)Marshal.SizeOf(typeof(A)); d.lp=new IntPtr(s); SHAppBarMessage(0xA, ref d); }
    public static void Ui(uint a, int v) { SystemParametersInfo(a, (uint)v, IntPtr.Zero, 0x03); }
    public static void Pv(uint a, int v) { SystemParametersInfo(a, 0, new IntPtr(v), 0x03); }
    public static void Anim(int v) { var ai=new ANIMATIONINFO(); ai.cbSize=(uint)Marshal.SizeOf(typeof(ANIMATIONINFO)); ai.iMinAnimate=v; SystemParametersInfo(0x0049, ai.cbSize, ref ai, 0x03); }
    public static void Cast(string s) { UIntPtr r; SendMessageTimeout(new IntPtr(0xFFFF), 0x1A, IntPtr.Zero, s, 2, 1000, out r); }
}
'@ -ErrorAction SilentlyContinue

$script:culprit = $null
function Check($phase) {
    Start-Sleep -Milliseconds 1200
    $a = [Bi]::Get()
    $on = [bool]($a -band 1)
    Write-Host ("  after {0,-42} ABM=0x{1:X2} autoHide={2,-5} {3}" -f $phase, $a, $on, $(if ($on) { 'still ON' } else { '<<< REVERTED' }))
    if (-not $on -and -not $script:culprit) { $script:culprit = $phase }
    return $on
}

function EnsureOn {
    if (-not ([Bi]::Get() -band 1)) { [Bi]::Set(1); Start-Sleep -Milliseconds 1200 }
}

Write-Host '=== bisecting the taskbar revert ==='
Write-Host ''
EnsureOn
Check 'baseline (ABM forced ON)' | Out-Null

# phase A: the SPI effect batch only
Write-Host "`n-- phase A: SPI effect writes (SPIF_UPDATEINIFILE|SPIF_SENDCHANGE) --"
EnsureOn
[Bi]::Ui(0x0025, 0)   # DragFullWindows
[Bi]::Ui(0x004B, 0)   # FontSmoothing
foreach ($a in 0x1003, 0x1005, 0x1007, 0x1015, 0x1017, 0x101B, 0x1025, 0x1043) { [Bi]::Pv([uint32]$a, 0) }
[Bi]::Anim(0)
Check 'phase A (SPI batch)' | Out-Null

# phase B: Explorer\Advanced registry writes
Write-Host "`n-- phase B: Explorer\Advanced writes --"
EnsureOn
$adv = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\Advanced'
Set-ItemProperty $adv -Name ListviewAlphaSelect -Value 0
Set-ItemProperty $adv -Name ListviewShadow -Value 0
Set-ItemProperty $adv -Name TaskbarAnimations -Value 0
Set-ItemProperty $adv -Name IconsOnly -Value 1
Check 'phase B (Advanced writes, no broadcast)' | Out-Null

# phase C: broadcast after the Advanced writes
Write-Host "`n-- phase C: broadcast after Advanced writes --"
EnsureOn
[Bi]::Cast('WindowMetrics')
Check 'phase C (WindowMetrics broadcast)' | Out-Null

Write-Host "`n-- phase D: TraySettings / Environment broadcasts --"
EnsureOn
[Bi]::Cast('TraySettings')
Check 'phase D (TraySettings broadcast)' | Out-Null

Write-Host "`n-- phase E: DWM writes + mask write --"
EnsureOn
Set-ItemProperty 'HKCU:\Software\Microsoft\Windows\DWM' -Name EnableAeroPeek -Value 0
Set-ItemProperty 'HKCU:\Software\Microsoft\Windows\DWM' -Name AlwaysHibernateThumbnails -Value 0
$cur = (Get-ItemProperty 'HKCU:\Control Panel\Desktop').UserPreferencesMask
Set-ItemProperty 'HKCU:\Control Panel\Desktop' -Name UserPreferencesMask -Value $cur
Check 'phase E (DWM + mask write)' | Out-Null

Write-Host "`n=== RESULT ==="
if ($script:culprit) { Write-Host "  first phase to revert the override: $($script:culprit)" }
else { Write-Host '  no single phase reverted it; the revert may need the full combination or be timing dependent' }

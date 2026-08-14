<#
.SYNOPSIS
  Phase 1 evidence: establishes a DURABLE taskbar auto-hide mechanism.

.DESCRIPTION
  Phase 0 proved ABM_SETSTATE changes the live state. This test additionally shows that
  the live state and the persisted StuckRects3 value can diverge, which lets Explorer
  silently revert the override later.

  Candidate fix under test: write BOTH.
    - ABM_SETSTATE      -> immediate live effect
    - StuckRects3 bit 0 -> durable agreement, so an Explorer reload cannot revert it

  Also covers the Phase 1 gate case that Phase 0 missed: a baseline of auto-hide OFF.
#>
$ErrorActionPreference = 'Stop'

Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class Td
{
    [StructLayout(LayoutKind.Sequential)]
    public struct A { public uint cbSize; public IntPtr hWnd; public uint m; public uint e; public int l,t,r,b; public IntPtr lp; }
    [DllImport("shell32.dll")] static extern IntPtr SHAppBarMessage(uint m, ref A d);
    [DllImport("user32.dll", CharSet=CharSet.Auto)] static extern IntPtr SendMessageTimeout(IntPtr h, uint m, IntPtr w, string l, uint f, uint t, out UIntPtr r);
    public static uint Get() { var d=new A(); d.cbSize=(uint)Marshal.SizeOf(typeof(A)); return (uint)SHAppBarMessage(4, ref d).ToInt64(); }
    public static void Set(uint s) { var d=new A(); d.cbSize=(uint)Marshal.SizeOf(typeof(A)); d.lp=new IntPtr(s); SHAppBarMessage(0xA, ref d); }
    public static void Cast(string s) { UIntPtr r; SendMessageTimeout(new IntPtr(0xFFFF), 0x1A, IntPtr.Zero, s, 2, 1000, out r); }
}
'@ -ErrorAction SilentlyContinue

$SRPath = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\StuckRects3'

function Get-SR { (Get-ItemProperty $SRPath).Settings }
function Get-SRAutoHide { [bool]((Get-SR)[8] -band 1) }

function Set-SRAutoHide([bool]$on) {
    $b = Get-SR
    if ($on) { $b[8] = $b[8] -bor 1 } else { $b[8] = $b[8] -band (-bnot 1) }
    Set-ItemProperty -LiteralPath $SRPath -Name Settings -Value $b
}

function State([string]$tag) {
    $a = [Td]::Get()
    $live = [bool]($a -band 1)
    $pers = Get-SRAutoHide
    $agree = ($live -eq $pers)
    Write-Host ("  {0,-34} live={1,-5} persisted={2,-5} {3}" -f $tag, $live, $pers, $(if ($agree) { 'agree' } else { '*** DIVERGED ***' }))
    return [pscustomobject]@{ Live = $live; Persisted = $pers; Agree = $agree; Raw = $a }
}

# durable setter: live + persisted together
function Set-AutoHideDurable([bool]$on) {
    $cur = [Td]::Get()
    $target = if ($on) { $cur -bor 1 } else { $cur -band (-bnot 1) }
    [Td]::Set([uint32]$target)     # live
    Set-SRAutoHide $on             # persisted
    Start-Sleep -Milliseconds 1000
}

$originalLive = [bool](([Td]::Get()) -band 1)
$originalPers = Get-SRAutoHide
Write-Host "=== taskbar durability ===`n"
Write-Host "OPERATOR ORIGINAL: live=$originalLive persisted=$originalPers  (will be restored to live=True/persisted=True at the end)`n"

try {
    State 'initial' | Out-Null

    Write-Host "`n--- test 1: durable setter reaches agreement (ON) ---"
    Set-AutoHideDurable $true
    $t1 = State 'after durable set ON'
    Write-Host "`n  -> broadcast, then re-check (this is what previously reverted it)"
    [Td]::Cast('WindowMetrics'); Start-Sleep -Milliseconds 1500
    $t1b = State 'after broadcast'
    $test1 = $t1.Agree -and $t1b.Live -and $t1b.Agree

    Write-Host "`n--- test 2: OFF baseline. apply override (OFF) then restore (OFF) ---"
    Write-Host '  simulating a user whose baseline is auto-hide OFF'
    Set-AutoHideDurable $false
    $base = State 'baseline captured = OFF'

    Write-Host "`n  -> Remotune apply: disable auto-hide (already OFF, must be a no-op)"
    Set-AutoHideDurable $false
    $applied = State 'after apply'

    Write-Host "`n  -> Remotune restore to the captured baseline (OFF)"
    Set-AutoHideDurable $base.Live
    [Td]::Cast('WindowMetrics'); Start-Sleep -Milliseconds 1200
    $restored = State 'after restore'

    $test2 = (-not $restored.Live) -and $restored.Agree
    Write-Host ''
    Write-Host "  OFF baseline preserved: $test2  (must stay OFF, never forced ON)"

    Write-Host "`n=== RESULT ==="
    Write-Host "  test 1 durable ON survives broadcast : $(if($test1){'PASS'}else{'FAIL'})"
    Write-Host "  test 2 OFF baseline round-trips      : $(if($test2){'PASS'}else{'FAIL'})"
}
finally {
    Write-Host "`n=== finally: restoring operator state (auto-hide ON, both layers) ==="
    Set-AutoHideDurable $true
    [Td]::Cast('WindowMetrics'); Start-Sleep -Milliseconds 1200
    State 'operator state restored' | Out-Null
}

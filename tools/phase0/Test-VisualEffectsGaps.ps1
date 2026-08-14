<#
.SYNOPSIS
  Closes the two remaining Phase 1 Visual Effects acceptance-gate cases.

.DESCRIPTION
  Gap A - baseline "Let Windows choose" (VisualFXSetting = 0):
      does writing 0 make Windows recompute and overwrite effect values,
      and does that baseline still round-trip exactly?

  Gap B - baseline already Best Performance:
      apply must be idempotent (a safe no-op) and restore must not invent changes.

  Always returns the machine to -OriginalLabel.
#>
param(
    [string]$OriginalLabel = '01-baseline-bestappearance'
)
$ErrorActionPreference = 'Stop'

$snapTool = Join-Path $PSScriptRoot 'Get-VisualState.ps1'
$restoreTool = Join-Path $PSScriptRoot 'Restore-VisualState.ps1'
$applyTool = Join-Path $PSScriptRoot 'Apply-BestPerformance.ps1'

function Snap([string]$l) { & $snapTool -Label $l | Out-Null }
function Restore([string]$l) { & $restoreTool -Label $l | Out-Null }
function ApplyBP { & $applyTool -Quiet | Out-Null }
function DiffText([string]$a, [string]$b) { (& $snapTool -Diff @($a, $b) | Out-String) }
function IsClean([string]$t) {
    # tolerate only the known FontSmoothingType schema artifact from pre-schema snapshots
    if ($t -match 'NO DIFFERENCES') { return $true }
    if (($t -match 'FontSmoothingType') -and ($t -match 'changed values: 1')) { return $true }
    return $false
}

$vfxPath = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Explorer\VisualEffects'
$results = @{}

try {
    # ============ GAP A ============
    Write-Host '=== GAP A: baseline "Let Windows choose" (VisualFXSetting = 0) ==='
    Restore $OriginalLabel
    $maskBefore = (($((Get-ItemProperty 'HKCU:\Control Panel\Desktop').UserPreferencesMask) | ForEach-Object { $_.ToString('X2') }) -join ' ')
    Write-Host "  mask before writing 0 : $maskBefore"

    Set-ItemProperty -LiteralPath $vfxPath -Name VisualFXSetting -Value 0
    Start-Sleep -Seconds 3
    $maskAfter = (($((Get-ItemProperty 'HKCU:\Control Panel\Desktop').UserPreferencesMask) | ForEach-Object { $_.ToString('X2') }) -join ' ')
    Write-Host "  mask after  writing 0 : $maskAfter"
    $drift = ($maskBefore -ne $maskAfter)
    Write-Host "  Windows recomputed/overwrote effect values: $drift"
    $results['A-drift'] = $drift

    Snap '10-letwindowschoose-baseline'
    Write-Host "`n  -> apply Best Performance over the 'Let Windows choose' baseline"
    ApplyBP
    Snap '11-bp-from-letwindowschoose'
    Write-Host "`n  -> restore the 'Let Windows choose' baseline"
    Restore '10-letwindowschoose-baseline'
    Snap '12-letwindowschoose-restored'
    $dA = DiffText '10-letwindowschoose-baseline' '12-letwindowschoose-restored'
    Write-Host $dA
    $results['A-roundtrip'] = IsClean $dA
    $vfxNow = (Get-ItemProperty $vfxPath).VisualFXSetting
    Write-Host "  VisualFXSetting after restore = $vfxNow (expect 0)"
    $results['A-vfx'] = ($vfxNow -eq 0)

    # ============ GAP B ============
    Write-Host "`n=== GAP B: baseline already Best Performance ==="
    ApplyBP
    Snap '13-bp-baseline'
    Write-Host '  -> apply Best Performance again (must be idempotent)'
    ApplyBP
    Snap '14-bp-applied-twice'
    $dB1 = DiffText '13-bp-baseline' '14-bp-applied-twice'
    Write-Host $dB1
    $results['B-idempotent'] = IsClean $dB1

    Write-Host '  -> restore the Best Performance baseline (must not invent changes)'
    Restore '13-bp-baseline'
    Snap '15-bp-restored'
    $dB2 = DiffText '13-bp-baseline' '15-bp-restored'
    Write-Host $dB2
    $results['B-restore'] = IsClean $dB2

    Write-Host "`n=== RESULTS ==="
    Write-Host "  A. 'Let Windows choose' round-trips exactly     : $(if($results['A-roundtrip'] -and $results['A-vfx']){'PASS'}else{'FAIL'})"
    Write-Host "     (Windows overwrote values on write of 0      : $($results['A-drift']))"
    Write-Host "  B. Best Performance baseline apply is idempotent: $(if($results['B-idempotent']){'PASS'}else{'FAIL'})"
    Write-Host "  B. Best Performance baseline restore is exact   : $(if($results['B-restore']){'PASS'}else{'FAIL'})"
}
finally {
    Write-Host "`n=== finally: restoring '$OriginalLabel' ==="
    Restore $OriginalLabel
    Snap '16-final'
    Write-Host (DiffText $OriginalLabel '16-final')
}

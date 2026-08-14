<#
.SYNOPSIS
  Phase 0 acceptance gate: proves an arbitrary Custom Visual Effects combination
  survives an Apply-Best-Performance / Restore cycle exactly.

.DESCRIPTION
  Sequence:
    1. synthesise an arbitrary Custom state that matches neither preset
    2. apply it, then capture what Windows actually holds
    3. apply Best Performance (the value set proven by the preset diff)
    4. restore the captured Custom state
    5. diff -> must be identical
    6. return the machine to the operator's original snapshot

  Always finishes by restoring -FinalLabel, even on failure.
#>
param(
    [string]$OriginalLabel = '01-baseline-bestappearance',
    [string]$BestPerfLabel = '02-preset-bestperformance',
    [string]$FinalLabel = '01-baseline-bestappearance'
)

$ErrorActionPreference = 'Stop'
$dir = Join-Path $PSScriptRoot 'snapshots'
$snapTool = Join-Path $PSScriptRoot 'Get-VisualState.ps1'
$restoreTool = Join-Path $PSScriptRoot 'Restore-VisualState.ps1'

function Snap([string]$label) { & $snapTool -Label $label | Out-Null }
function Apply([string]$label) { & $restoreTool -Label $label | Out-Null }
function DiffOf([string]$a, [string]$b) { (& $snapTool -Diff @($a, $b)) -join "`n" }

try {
    # ---- step 1: synthesise an arbitrary Custom state ----
    Write-Host '=== step 1: synthesise arbitrary Custom state ==='
    $c = Get-Content (Join-Path $dir "$OriginalLabel.json") -Raw | ConvertFrom-Json

    $custom = @{
        MenuAnimation = 0; ComboBoxAnimation = 1; ListBoxSmoothScrolling = 0
        GradientCaptions = 0; KeyboardCues = 1; HotTracking = 0
        MenuFade = 0; SelectionFade = 1; TooltipAnimation = 0; TooltipFade = 0
        CursorShadow = 0; DropShadow = 1; UIEffects = 1; ClientAreaAnimation = 0
        DragFullWindows = 0; MinAnimate = 0
    }
    foreach ($k in $custom.Keys) { $c.spi.$k = $custom[$k] }
    $c.registry.VisualFXSetting = 3
    $c.registry.'Advanced.IconsOnly' = 1
    $c.registry.'Advanced.ListviewShadow' = 0
    $c.registry.'Advanced.TaskbarAnimations' = 0
    $c.registry.'DWM.EnableAeroPeek' = 0
    $c.registry.'WindowMetrics.MinAnimate' = '0'
    $c.registry.'Desktop.DragFullWindows' = '0'

    # mask consistent with the per-effect values above, unattributed bits deliberately kept
    #   byte0: keep 0x04 (ComboBox) only            -> 0x04
    #   byte1: keep 0x04 (SelectionFade) only        -> 0x04
    #   byte2: keep 0x07 verbatim (incl. unattributed 0x04)
    #   byte3: keep 0x80 (UIEffects)
    #   byte4: clear 0x02 (ClientAreaAnimation), keep unattributed 0x10
    $c.registry.UserPreferencesMask = '0x0404078010000000'
    $c | ConvertTo-Json -Depth 8 | Set-Content (Join-Path $dir '04-custom-target.json') -Encoding utf8
    Write-Host "  synthesised: VisualFXSetting=3 mask=$($c.registry.UserPreferencesMask)"

    # ---- step 2: apply it and capture the real resulting state ----
    Write-Host "`n=== step 2: apply Custom, capture actual ==="
    Apply '04-custom-target'
    Snap '04-custom-actual'
    $a = Get-Content (Join-Path $dir '04-custom-actual.json') -Raw | ConvertFrom-Json
    Write-Host "  actual VisualFXSetting=$($a.registry.VisualFXSetting) mask=$($a.registry.UserPreferencesMask)"

    # ---- step 3: apply Best Performance over the Custom state ----
    Write-Host "`n=== step 3: apply Best Performance over Custom ==="
    Apply $BestPerfLabel
    Snap '05-bestperf-from-custom'
    Write-Host '  diff vs the operator-produced Best Performance state:'
    Write-Host (DiffOf $BestPerfLabel '05-bestperf-from-custom')

    # ---- step 4/5: restore the Custom state and compare ----
    Write-Host "`n=== step 4: restore Custom, then diff (ACCEPTANCE GATE) ==="
    Apply '04-custom-actual'
    Snap '06-custom-restored'
    $gate = DiffOf '04-custom-actual' '06-custom-restored'
    Write-Host $gate
    $pass = $gate -match 'NO DIFFERENCES'
    Write-Host ''
    Write-Host "  CUSTOM ROUND-TRIP: $(if($pass){'PASS'}else{'FAIL'})"
}
finally {
    Write-Host "`n=== finally: returning machine to '$FinalLabel' ==="
    Apply $FinalLabel
    Snap '07-final-state'
    Write-Host (DiffOf $FinalLabel '07-final-state')
}

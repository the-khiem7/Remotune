<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { AdoptCurrentWindowsVisualEffects, GetAutostartStatus, GetProfileSettings, OpenWindowsPerformanceOptions, Pause, PortablePathStatus, RestoreNow, Resume, SetAutostart, SetProfileSettings, Status } from './wails'

type CoordinatorStatus = {
  Tuning: number
  CRD: number
  AutomationEnabled: boolean
  Paused: boolean
  Owned: boolean
  Detector: {
    Health: string
    LastTransition: string
    LastTransitionAt: string
    LastRecordID: number
    LastPollError: string
    ConsecutivePollErrors: number
    SkippedRecords: number
    LastReconciledAt: string
  }
}

type AutostartStatus = {
  Registered: boolean
  PathMatch: boolean
  RegisteredPath: string
}

type PortablePathStatus = {
  AutostartRegistered: boolean
  PathMismatch: boolean
  RegisteredPath: string
  CurrentPath: string
}

type ProfileSettings = {
  schemaVersion: number
  crdOnProfile: string
  crdOffAction: string
  customEffects: Record<string, boolean>
}

const crdLabels = ['Unknown', 'Disconnected', 'Connected']
const tuningLabels = ['Unknown', 'Baseline', 'Applying', 'Active', 'Restoring', 'Partial/Error', 'Recovery Required']

const status = ref<CoordinatorStatus | null>(null)
const autostart = ref<AutostartStatus | null>(null)
const portablePath = ref<PortablePathStatus | null>(null)
const profiles = ref<ProfileSettings | null>(null)
const busy = ref(false)
const error = ref('')
const notice = ref('')
let refreshTimer: number | undefined

const automationLabel = computed(() => status.value?.Paused ? 'Paused' : 'Enabled')
const automationTone = computed(() => status.value?.Paused ? 'muted' : 'success')
const autostartEnabled = computed(() => Boolean(autostart.value?.Registered && autostart.value.PathMatch))
const tuningLabel = computed(() => tuningLabels[status.value?.Tuning ?? 0] ?? 'Unknown')
const crdLabel = computed(() => crdLabels[status.value?.CRD ?? 0] ?? 'Unknown')
const detectorLabel = computed(() => status.value?.Detector?.Health ?? 'Starting')
const pauseHint = computed(() => status.value?.Paused
  ? 'Automation is paused. Resume to react to the current CRD state.'
  : status.value?.Owned
    ? 'Pause restores the Remotune-owned baseline, then stops automation.'
    : 'Pause stops automation. There is no Remotune-owned snapshot to restore yet.')
const profileLabel = computed(() => ({ windowsChoose: 'Let Windows choose', bestAppearance: 'Best Appearance', bestPerformance: 'Best Performance', custom: 'Custom' }[profiles.value?.crdOnProfile ?? ''] ?? 'Loading'))

async function refresh(clearError = true) {
  try {
    const [nextStatus, nextAutostart, nextPortablePath, nextProfiles] = await Promise.all([Status(), GetAutostartStatus(), PortablePathStatus(), GetProfileSettings()])
    status.value = nextStatus as CoordinatorStatus
    autostart.value = nextAutostart as AutostartStatus
    portablePath.value = nextPortablePath as PortablePathStatus
    profiles.value = nextProfiles as ProfileSettings
    if (clearError) error.value = ''
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  }
}

async function run(command: () => Promise<unknown>, successNotice = '') {
  busy.value = true
  error.value = ''
  notice.value = ''
  let commandError = ''
  try {
    await command()
    notice.value = successNotice
  } catch (cause) {
    commandError = cause instanceof Error ? cause.message : String(cause)
  } finally {
    busy.value = false
    await refresh(false)
    if (commandError) error.value = commandError
  }
}

function toggleAutomation() {
  const pausing = !status.value?.Paused
  const hasSnapshot = Boolean(status.value?.Owned)
  return run(
    () => pausing ? Pause() : Resume(),
    pausing
      ? hasSnapshot ? 'Automation paused and the Remotune-owned baseline was restored.' : 'Automation paused. No Remotune-owned snapshot needed restoration.'
      : 'Automation resumed and reconciled with the current CRD state.'
  )
}

function restoreNow() {
  return run(() => RestoreNow(), 'Remotune-owned baseline restored.')
}

function toggleAutostart() {
  return run(() => SetAutostart(!autostartEnabled.value), 'Start with Windows updated.')
}

function updateProfiles(update: Partial<ProfileSettings>, successNotice: string) {
  if (!profiles.value) return Promise.resolve()
  const next = {
    ...profiles.value,
    ...update,
    customEffects: { ...profiles.value.customEffects, ...(update.customEffects ?? {}) },
  }
  return run(async () => { profiles.value = await SetProfileSettings(next as Parameters<typeof SetProfileSettings>[0]) as ProfileSettings }, successNotice)
}

function selectOnProfile(profile: string) {
  return updateProfiles({ crdOnProfile: profile }, 'CRD-on Visual Effects profile updated.')
}

function selectOffAction(action: string) {
  return updateProfiles({ crdOffAction: action }, 'CRD-off action updated.')
}

function openWindowsPerformanceOptions() { return run(() => OpenWindowsPerformanceOptions(), 'Windows Performance Options opened. Apply your changes there, then use the current settings here.') }
function adoptCurrentWindowsSettings() { return run(async () => { profiles.value = await AdoptCurrentWindowsVisualEffects() as ProfileSettings }, 'Current Windows Visual Effects saved as Remotune Custom.') }

onMounted(async () => {
  await refresh()
  refreshTimer = window.setInterval(refresh, 1500)
})

onUnmounted(() => {
  if (refreshTimer !== undefined) window.clearInterval(refreshTimer)
})
</script>

<template>
  <main class="panel" aria-live="polite">
    <header class="header">
      <h1>Remotune</h1>
      <button class="switch" :aria-label="automationLabel" :aria-pressed="!status?.Paused" :disabled="busy" @click="toggleAutomation"><span></span></button>
    </header>

    <section class="connection" :class="crdLabel.toLowerCase()">
      <span class="crd-icon" aria-hidden="true">◉</span>
      <strong>Chrome Remote Desktop</strong>
      <span class="connection-pill">{{ crdLabel }}</span>
    </section>

    <section class="card" aria-labelledby="automation-heading">
      <div class="section-heading">
        <div>
          <h2 id="automation-heading">Automatic tuning</h2>
          <p>{{ pauseHint }}</p>
        </div>
      </div>
      <div class="detail-row"><span>Visual Effects</span><strong>{{ profileLabel }} during CRD</strong></div>
      <div class="detail-row"><span>Taskbar auto-hide</span><strong>Off during CRD</strong></div>
    </section>

    <section class="card profiles" aria-labelledby="profiles-heading">
      <div class="section-heading compact"><div><h2 id="profiles-heading">Visual Effects profiles</h2><p>Profiles are applied and verified by Windows, not inferred by this UI.</p></div></div>
      <fieldset class="profile-group" :disabled="busy || !profiles"><legend>CRD ON</legend>
        <label v-for="option in [{ value: 'windowsChoose', label: 'Let Windows choose' }, { value: 'bestAppearance', label: 'Best Appearance' }, { value: 'bestPerformance', label: 'Best Performance' }, { value: 'custom', label: 'Custom' }]" :key="option.value" class="choice"><input type="radio" name="crd-on" :checked="profiles?.crdOnProfile === option.value" @change="selectOnProfile(option.value)"><span>{{ option.label }}</span></label>
      </fieldset>
      <fieldset class="profile-group" :disabled="busy || !profiles"><legend>CRD OFF</legend>
        <label v-for="option in [{ value: 'restoreSnapshot', label: 'Revert to snapshot' }, { value: 'windowsChoose', label: 'Let Windows choose' }, { value: 'bestAppearance', label: 'Best Appearance' }, { value: 'bestPerformance', label: 'Best Performance' }]" :key="option.value" class="choice"><input type="radio" name="crd-off" :checked="profiles?.crdOffAction === option.value" @change="selectOffAction(option.value)"><span>{{ option.label }}</span></label>
      </fieldset>
      <div v-if="profiles?.crdOnProfile === 'custom'" class="native-effects">
        <p>Use the Windows editor for the exact Visual Effects UI and behavior.</p>
        <div class="native-actions"><button class="secondary" :disabled="busy" @click="openWindowsPerformanceOptions">Open Windows Performance Options</button><button class="text-button" :disabled="busy" @click="adoptCurrentWindowsSettings">Use current Windows settings</button></div>
      </div>
    </section>

    <section class="card" aria-labelledby="state-heading">
      <div class="section-heading compact">
        <div>
          <h2 id="state-heading">Current state</h2>
          <p>Remotune keeps your original state until the selected action completes safely.</p>
        </div>
      </div>
      <div class="state-grid">
        <div><span>Tuning</span><strong>{{ tuningLabel }}</strong></div>
        <div><span>Recovery snapshot</span><strong>{{ status?.Owned ? 'Available' : 'None' }}</strong></div>
      </div>
      <p class="detector-status" :class="detectorLabel.toLowerCase()">CRD detector: {{ detectorLabel }}<template v-if="status?.Detector?.LastTransition"> · last event {{ status.Detector.LastTransition }} #{{ status.Detector.LastRecordID }}</template></p>
      <p v-if="status?.Detector?.LastPollError" class="warning">Detector will reconcile automatically. {{ status.Detector.LastPollError }}</p>
    </section>

    <section class="card settings" aria-labelledby="startup-heading">
      <div>
        <h2 id="startup-heading">Start with Windows</h2>
        <p v-if="portablePath?.AutostartRegistered && portablePath.PathMismatch" class="warning">The startup entry points to a moved executable.</p>
        <p v-else>Launch Remotune when you sign in.</p>
      </div>
      <button class="switch" :aria-pressed="autostartEnabled" :disabled="busy" @click="toggleAutostart">
        <span></span>
      </button>
    </section>

    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <p v-if="notice" class="notice" role="status">{{ notice }}</p>

    <footer class="actions">
      <button class="secondary" :disabled="busy || !status?.Owned" @click="restoreNow">Restore Now</button>
      <button class="text-button" :disabled="busy" @click="() => refresh()">Refresh status</button>
    </footer>
  </main>
</template>

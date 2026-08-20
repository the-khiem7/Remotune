<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { GetAutostartStatus, Pause, PortablePathStatus, RestoreNow, Resume, SetAutostart, Status } from './wails'

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

const crdLabels = ['Unknown', 'Disconnected', 'Connected']
const tuningLabels = ['Unknown', 'Baseline', 'Applying', 'Active', 'Restoring', 'Partial/Error', 'Recovery Required']

const status = ref<CoordinatorStatus | null>(null)
const autostart = ref<AutostartStatus | null>(null)
const portablePath = ref<PortablePathStatus | null>(null)
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

async function refresh(clearError = true) {
  try {
    const [nextStatus, nextAutostart, nextPortablePath] = await Promise.all([Status(), GetAutostartStatus(), PortablePathStatus()])
    status.value = nextStatus as CoordinatorStatus
    autostart.value = nextAutostart as AutostartStatus
    portablePath.value = nextPortablePath as PortablePathStatus
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
      <div>
        <p class="eyebrow">REMOTE SESSION UTILITY</p>
        <h1>Remotune</h1>
      </div>
      <span class="status-pill" :class="automationTone">{{ automationLabel }}</span>
    </header>

    <section class="connection" :class="crdLabel.toLowerCase()">
      <span class="indicator" aria-hidden="true"></span>
      <div>
        <p>Chrome Remote Desktop</p>
        <strong>{{ crdLabel }}</strong>
      </div>
    </section>

    <section class="card" aria-labelledby="automation-heading">
      <div class="section-heading">
        <div>
          <h2 id="automation-heading">Automatic tuning</h2>
          <p>{{ pauseHint }}</p>
        </div>
        <button class="switch" :aria-pressed="!status?.Paused" :disabled="busy" @click="toggleAutomation">
          <span></span>
        </button>
      </div>
      <div class="detail-row"><span>Visual Effects</span><strong>Best Performance during CRD</strong></div>
      <div class="detail-row"><span>Taskbar auto-hide</span><strong>Off during CRD</strong></div>
    </section>

    <section class="card" aria-labelledby="state-heading">
      <div class="section-heading compact">
        <div>
          <h2 id="state-heading">Current state</h2>
          <p>Read from the coordinator, not inferred by the UI.</p>
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

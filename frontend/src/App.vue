<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { GetAutostartStatus, Pause, PortablePathStatus, RestoreNow, Resume, SetAutostart, Status } from './wails'

type CoordinatorStatus = {
  Tuning: string
  CRD: string
  AutomationEnabled: boolean
  Paused: boolean
  Owned: boolean
}

type AutostartStatus = {
  Registered: boolean
  PathMatch: boolean
  RegisteredPath: string
}

type PortablePathStatus = {
  AutostartRegistered: boolean
  PathMatches: boolean
  RegisteredPath: string
  CurrentPath: string
}

const status = ref<CoordinatorStatus | null>(null)
const autostart = ref<AutostartStatus | null>(null)
const portablePath = ref<PortablePathStatus | null>(null)
const busy = ref(false)
const error = ref('')
let refreshTimer: number | undefined

const automationLabel = computed(() => status.value?.Paused ? 'Paused' : 'Enabled')
const automationTone = computed(() => status.value?.Paused ? 'muted' : 'success')
const autostartEnabled = computed(() => Boolean(autostart.value?.Registered && autostart.value.PathMatch))
const tuningLabel = computed(() => status.value?.Tuning ?? 'Starting')
const crdLabel = computed(() => status.value?.CRD ?? 'Unknown')

async function refresh() {
  try {
    const [nextStatus, nextAutostart, nextPortablePath] = await Promise.all([Status(), GetAutostartStatus(), PortablePathStatus()])
    status.value = nextStatus as CoordinatorStatus
    autostart.value = nextAutostart as AutostartStatus
    portablePath.value = nextPortablePath as PortablePathStatus
    error.value = ''
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  }
}

async function run(command: () => Promise<unknown>) {
  busy.value = true
  error.value = ''
  try {
    await command()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    busy.value = false
    await refresh()
  }
}

function toggleAutomation() {
  return run(() => status.value?.Paused ? Resume() : Pause())
}

function restoreNow() {
  return run(() => RestoreNow())
}

function toggleAutostart() {
  return run(() => SetAutostart(!autostartEnabled.value))
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
          <p>Windows changes are applied only while Remotune owns a session baseline.</p>
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
    </section>

    <section class="card settings" aria-labelledby="startup-heading">
      <div>
        <h2 id="startup-heading">Start with Windows</h2>
        <p v-if="portablePath?.AutostartRegistered && !portablePath.PathMatches" class="warning">The startup entry points to a moved executable.</p>
        <p v-else>Launch Remotune when you sign in.</p>
      </div>
      <button class="switch" :aria-pressed="autostartEnabled" :disabled="busy" @click="toggleAutostart">
        <span></span>
      </button>
    </section>

    <p v-if="error" class="error" role="alert">{{ error }}</p>

    <footer class="actions">
      <button class="secondary" :disabled="busy || !status?.Owned" @click="restoreNow">Restore Now</button>
      <button class="text-button" :disabled="busy" @click="refresh">Refresh status</button>
    </footer>
  </main>
</template>

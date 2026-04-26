<template>
  <div class="filter-bar">
    <div class="filter-group">
      <span class="filter-label">Protocol</span>
      <div class="button-group">
        <button
          v-for="opt in protoOptions"
          :key="opt.value"
          :class="['filter-btn', { active: protoFilter === opt.value }]"
          @click="store.setProtoFilter(opt.value)"
        >
          {{ opt.label }}
        </button>
      </div>
    </div>

    <div class="filter-group">
      <span class="filter-label">IP Version</span>
      <div class="button-group">
        <button
          v-for="opt in ipVerOptions"
          :key="opt.value"
          :class="['filter-btn', { active: ipVerFilter === opt.value }]"
          @click="store.setIPVerFilter(opt.value)"
        >
          {{ opt.label }}
        </button>
      </div>
    </div>

    <div class="filter-group" :class="{ disabled: !hasContainerData }">
      <span class="filter-label">Docker</span>
      <div class="button-group">
        <button
          v-for="opt in containerOptions"
          :key="opt.value"
          :class="['filter-btn', { active: containerFilter === opt.value }]"
          @click="hasContainerData && store.setContainerFilter(opt.value)"
          :disabled="!hasContainerData"
        >
          {{ opt.label }}
        </button>
      </div>
      <span v-if="!hasContainerData && dockerError" class="filter-error" :title="dockerError">
        🐳 {{ dockerError }}
      </span>
      <span v-else-if="!hasContainerData" class="filter-hint" title="Mount docker.sock and set DOCKER_HOST in docker-compose.yml">
        🐳 off
      </span>
    </div>

    <div class="filter-group">
      <button class="freeze-btn" :class="{ active: props.isFrozen }" @click="emit('toggle-freeze')" :title="props.isFrozen ? 'Resume live updates' : 'Pause live updates'">
        {{ props.isFrozen ? '▶' : '⏸' }}
      </button>
    </div>

    <div class="filter-group">
      <button class="export-btn" @click="exportCSV" title="Export as CSV">
        ↓ CSV
      </button>
      <button class="export-btn" @click="exportJSON" title="Export as JSON">
        ↓ JSON
      </button>
    </div>
  </div>
</template>

<script setup>
import { useSocketsStore } from '../stores/sockets'
import { storeToRefs } from 'pinia'

const props = defineProps({
  isFrozen: { type: Boolean, default: false }
})
const emit = defineEmits(['toggle-freeze'])

const store = useSocketsStore()
const { protoFilter, ipVerFilter, containerFilter, hasContainerData, dockerError } = storeToRefs(store)

const protoOptions = [
  { label: 'TCP', value: 'tcp' },
  { label: 'UDP', value: 'udp' },
  { label: 'Both', value: 'both' }
]

const ipVerOptions = [
  { label: 'IPv4', value: '4' },
  { label: 'IPv6', value: '6' },
  { label: 'Both', value: 'both' }
]

const containerOptions = [
  { label: 'All', value: 'all' },
  { label: 'Docker', value: 'with' },
  { label: 'No Docker', value: 'without' }
]

function csvEscape(field) {
  const str = String(field ?? '')
  if (str.includes(',') || str.includes('"') || str.includes('\n')) {
    return '"' + str.replace(/"/g, '""') + '"'
  }
  return str
}

function getExportData() {
  return store.filteredAndSortedSockets.filter(item => item._type === 'socket')
}

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

function exportCSV() {
  const data = getExportData()
  const headers = ['protocol', 'local_addr', 'local_port', 'remote_addr', 'remote_port', 'state', 'process', 'command', 'container', 'image']
  const rows = [headers.map(csvEscape).join(',')]

  for (const item of data) {
    rows.push([
      csvEscape(item.protocol),
      csvEscape(item.local_addr),
      csvEscape(item.local_port),
      csvEscape(item.remote_addr),
      csvEscape(item.remote_port),
      csvEscape(item.state),
      csvEscape(item.process),
      csvEscape(item.command),
      csvEscape(item.container),
      csvEscape(item.c_image)
    ].join(','))
  }

  const blob = new Blob([rows.join('\n')], { type: 'text/csv;charset=utf-8;' })
  downloadBlob(blob, `ports-${Date.now()}.csv`)
}

function exportJSON() {
  const data = getExportData().map(item => ({
    protocol: item.protocol,
    local_addr: item.local_addr,
    local_port: item.local_port,
    remote_addr: item.remote_addr,
    remote_port: item.remote_port,
    state: item.state,
    process: item.process,
    command: item.command || null,
    exe: item.exe || null,
    container: item.container || null,
    image: item.c_image || null,
    network: item.c_network || null
  }))

  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json;charset=utf-8;' })
  downloadBlob(blob, `ports-${Date.now()}.json`)
}
</script>

<style scoped>
.filter-bar {
  display: flex;
  gap: 32px;
  padding: 12px 16px;
  background-color: #1a1a2e;
  border-bottom: 1px solid #2d2d44;
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 12px;
}

.filter-label {
  font-family: 'Courier New', monospace;
  font-size: 11px;
  font-weight: 600;
  color: #8080a0;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.button-group {
  display: flex;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid #2d2d44;
}

.filter-btn {
  padding: 6px 14px;
  border: none;
  border-right: 1px solid #2d2d44;
  background-color: #252540;
  color: #a0a0c0;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.15s ease, color 0.15s ease;
}

.filter-btn:last-child {
  border-right: none;
}

.filter-btn:hover:not(.active) {
  background-color: #2d2d50;
  color: #c0c0e0;
}

.filter-btn.active {
  background-color: #3d3d6b;
  color: #e0e0ff;
  font-weight: 600;
}

.filter-group.disabled {
  opacity: 0.5;
}

.filter-group.disabled .filter-btn {
  cursor: not-allowed;
}

.filter-hint {
  font-size: 10px;
  color: #7c3aed;
  margin-left: 8px;
  cursor: help;
}

.filter-error {
  font-size: 10px;
  color: #ef4444;
  margin-left: 8px;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: help;
}

.freeze-btn {
  padding: 6px 12px;
  border: 1px solid #2d2d44;
  border-radius: 6px;
  background-color: #252540;
  color: #a0a0c0;
  font-size: 14px;
  cursor: pointer;
  transition: background-color 0.15s ease, color 0.15s ease;
  line-height: 1;
}

.freeze-btn:hover {
  background-color: #2d2d50;
  color: #c0c0e0;
}

.freeze-btn.active {
  background-color: #7c3aed;
  color: #ffffff;
  border-color: #7c3aed;
}

.export-btn {
  padding: 6px 12px;
  border: 1px solid #2d2d44;
  border-radius: 6px;
  background-color: #252540;
  color: #a0a0c0;
  font-family: 'Courier New', monospace;
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.15s ease, color 0.15s ease;
}

.export-btn:hover {
  background-color: #2d2d50;
  color: #c0c0e0;
}
</style>

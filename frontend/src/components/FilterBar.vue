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
      <span v-if="!hasContainerData" class="filter-hint" title="Mount docker.sock and set DOCKER_HOST in docker-compose.yml">
        🐳 off
      </span>
    </div>
  </div>
</template>

<script setup>
import { useSocketsStore } from '../stores/sockets'
import { storeToRefs } from 'pinia'

const store = useSocketsStore()
const { protoFilter, ipVerFilter, containerFilter, hasContainerData } = storeToRefs(store)

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
</style>

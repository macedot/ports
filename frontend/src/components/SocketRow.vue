<template>
  <div class="socket-row">
    <span class="col-protocol">
      <span class="badge" :class="protocolClass">{{ item.protocol }}</span>
    </span>
    <span class="col-local">{{ grouped ? item.local_addr : item.local_addr + ':' + item.local_port }}</span>
    <span class="col-foreign">{{ item.remote_addr }}:{{ item.remote_port }}</span>
    <span class="col-state">
      <span class="badge" :class="stateClass">{{ item.state }}</span>
    </span>
    <span class="col-process">{{ displayProcess }}</span>
    <span class="col-container">
      <span v-if="item.container" class="container-badge" :title="item.c_image">
        {{ displayContainer }}
      </span>
      <span v-else class="container-empty">—</span>
    </span>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  item: {
    type: Object,
    required: true
  },
  grouped: {
    type: Boolean,
    default: false
  }
})

const protocolClass = computed(() => {
  const p = props.item.protocol?.toUpperCase() ?? ''
  if (p === 'TCP') return 'proto-tcp'
  if (p === 'TCP6') return 'proto-tcp6'
  if (p === 'UDP') return 'proto-udp'
  if (p === 'UDP6') return 'proto-udp6'
  return 'proto-default'
})

const stateClass = computed(() => {
  const s = props.item.state?.toUpperCase() ?? ''
  if (s === 'LISTEN') return 'state-listen'
  if (s === 'ESTABLISHED') return 'state-established'
  if (s === 'TIME_WAIT') return 'state-timewait'
  return 'state-default'
})

const displayProcess = computed(() => {
  return props.item.process?.trim() || '?'
})

const displayContainer = computed(() => {
  return props.item.container?.trim() || '—'
})
</script>

<style scoped>
.socket-row {
  display: grid;
  grid-template-columns: 60px 1fr 1fr 100px 140px 160px;
  align-items: center;
  height: 32px;
  padding: 0 12px;
  background-color: #1a1a2e;
  border-bottom: 1px solid #2d2d44;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  color: #e0e0e0;
}

.col-protocol {
  display: flex;
  align-items: center;
}

.col-local,
.col-foreign {
  font-family: 'Courier New', monospace;
  font-size: 12px;
  color: #a0a0c0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.col-state {
  display: flex;
  align-items: center;
}

.col-process {
  font-family: 'Courier New', monospace;
  font-size: 12px;
  color: #c0c0c0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.col-container {
  font-family: 'Courier New', monospace;
  font-size: 11px;
  color: #c0c0c0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-badge {
  display: inline-block;
  padding: 2px 6px;
  border-radius: 3px;
  background-color: #7c3aed;
  color: #ffffff;
  font-size: 10px;
  font-weight: 600;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
}

.container-empty {
  color: #4a4a6a;
}

.badge {
  display: inline-block;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
}

/* Protocol colors */
.proto-tcp {
  background-color: #2563eb;
  color: #ffffff;
}

.proto-tcp6 {
  background-color: #06b6d4;
  color: #000000;
}

.proto-udp {
  background-color: #16a34a;
  color: #ffffff;
}

.proto-udp6 {
  background-color: #84cc16;
  color: #000000;
}

.proto-default {
  background-color: #6b7280;
  color: #ffffff;
}

/* State colors */
.state-listen {
  background-color: #16a34a;
  color: #ffffff;
}

.state-established {
  background-color: #2563eb;
  color: #ffffff;
}

.state-timewait {
  background-color: #ea580c;
  color: #ffffff;
}

.state-default {
  background-color: #6b7280;
  color: #ffffff;
}
</style>
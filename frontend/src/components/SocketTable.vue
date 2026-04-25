<template>
  <div class="socket-table-container">
    <div class="table-header">
      <div class="header-row">
        <div class="header-cell sortable" @click="store.toggleSort('protocol')">
          <span>Protocol</span>
          <span class="sort-indicator">{{ getSortIndicator('protocol') }}</span>
        </div>
        <div class="header-cell sortable" @click="store.toggleSort('local_port')">
          <span>Local Port</span>
          <span class="sort-indicator">{{ getSortIndicator('local_port') }}</span>
        </div>
        <div class="header-cell">
          <span>Foreign Address</span>
        </div>
        <div class="header-cell sortable" @click="store.toggleSort('state')">
          <span>State</span>
          <span class="sort-indicator">{{ getSortIndicator('state') }}</span>
        </div>
        <div class="header-cell sortable" @click="store.toggleSort('process')">
          <span>Process</span>
          <span class="sort-indicator">{{ getSortIndicator('process') }}</span>
        </div>
      </div>
    </div>

    <div class="table-body">
      <RecycleScroller
        v-if="socketItemCount > 0"
        :items="filteredAndSortedSockets"
        :item-size="32"
        key-field="_key"
        v-slot="{ item }"
      >
        <div v-if="item._type === 'group'" class="group-header">
          <span class="port-label">Port {{ item.port }}</span>
          <span class="count-badge">{{ item.count }} connection{{ item.count !== 1 ? 's' : '' }}</span>
        </div>
        <SocketRow v-else :item="item" :grouped="true" />
      </RecycleScroller>
      <div v-else class="empty-state">
        No sockets found
      </div>
    </div>

    <div class="table-footer">
      <span class="socket-count">{{ socketItemCount }} socket{{ socketItemCount !== 1 ? 's' : '' }}</span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { RecycleScroller } from 'vue-virtual-scroller'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'
import SocketRow from './SocketRow.vue'
import { useSocketsStore } from '../stores/sockets'

const store = useSocketsStore()
const { filteredAndSortedSockets, sortKey, sortDir } = storeToRefs(store)

const socketItemCount = computed(() => {
  const items = filteredAndSortedSockets.value
  let count = 0
  for (const item of items) {
    if (item._type === 'socket') count++
  }
  return count
})

const getSortIndicator = (key) => {
  if (sortKey.value !== key) return ''
  return sortDir.value === 'asc' ? '▲' : '▼'
}
</script>

<style scoped>
.socket-table-container {
  display: flex;
  flex-direction: column;
  flex: 1;
  overflow: hidden;
  background-color: #0f0f1a;
  color: #e0e0e0;
}

.table-header {
  flex-shrink: 0;
  background-color: #1a1a2e;
  border-bottom: 1px solid #2d2d44;
}

.header-row {
  display: grid;
  grid-template-columns: 60px 1fr 1fr 100px 140px;
  align-items: center;
  height: 36px;
  padding: 0 12px;
  font-family: 'Courier New', monospace;
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  color: #8080a0;
}

.header-cell {
  display: flex;
  align-items: center;
  gap: 4px;
}

.header-cell.sortable {
  cursor: pointer;
  user-select: none;
}

.header-cell.sortable:hover {
  color: #c0c0e0;
}

.sort-indicator {
  font-size: 10px;
  color: #4a4a6a;
}

.table-body {
  flex: 1;
  overflow: hidden;
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-family: 'Courier New', monospace;
  font-size: 14px;
  color: #606080;
}

.table-footer {
  flex-shrink: 0;
  padding: 6px 12px;
  background-color: #1a1a2e;
  border-top: 1px solid #2d2d44;
  font-family: 'Courier New', monospace;
  font-size: 11px;
  color: #606080;
}

.socket-count {
  color: #8080a0;
}

.group-header {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  gap: 10px;
  height: 32px;
  padding: 0 12px;
  background: #1e1e36;
  border-left: 3px solid #4a9eff;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  color: #c0c0e0;
}

.group-header .port-label {
  font-weight: 700;
  color: #e0e0ff;
  font-size: 13px;
}

.group-header .count-badge {
  font-size: 10px;
  color: #8888aa;
  background: #2d2d44;
  padding: 2px 6px;
  border-radius: 3px;
}
</style>

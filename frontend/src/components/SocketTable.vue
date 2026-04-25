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
        v-if="filteredAndSortedSockets.length > 0"
        :items="filteredAndSortedSockets"
        :item-size="32"
        key-field="_key"
        v-slot="{ item }"
      >
        <SocketRow :item="item" />
      </RecycleScroller>
      <div v-else class="empty-state">
        No sockets found
      </div>
    </div>

    <div class="table-footer">
      <span class="socket-count">{{ filteredAndSortedSockets.length }} socket{{ filteredAndSortedSockets.length !== 1 ? 's' : '' }}</span>
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
</style>

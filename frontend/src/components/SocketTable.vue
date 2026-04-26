<template>
  <div class="socket-table-container">
    <div class="table-header">
      <div class="search-row">
        <span class="search-icon">⌕</span>
        <input
          type="text"
          class="search-input"
          placeholder="Regex filter (e.g. :80|nginx|LISTEN)..."
          :value="store.searchQuery"
          @input="store.setSearchQuery($event.target.value)"
        />
        <span v-if="store.searchError" class="search-error">{{ store.searchError }}</span>
      </div>
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
        <div class="header-cell sortable" @click="store.toggleSort('container')">
          <span>Container</span>
          <span class="sort-indicator">{{ getSortIndicator('container') }}</span>
        </div>
      </div>
      <button class="fold-all-btn" @click="toggleAll">
        {{ isAllCollapsed ? 'Expand All' : 'Collapse All' }}
      </button>
    </div>

    <div class="table-body">
      <RecycleScroller
        v-if="filteredAndSortedSockets.length > 0"
        :items="filteredAndSortedSockets"
        :item-size="32"
        key-field="_key"
        v-slot="{ item }"
      >
        <div v-if="item._type === 'group'" class="group-header" @click="store.toggleGroup(item.port)">
          <span class="chevron">{{ collapsedGroups.has(item.port) ? '▸' : '▾' }}</span>
          <span class="port-label">Port {{ item.port }}</span>
          <span class="count-badge">{{ item.count }} connection{{ item.count !== 1 ? 's' : '' }}</span>
          <span v-if="item.container" class="container-group-badge" :title="item.c_image">
            {{ item.container }}
          </span>
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
const { filteredAndSortedSockets, sortKey, sortDir, collapsedGroups } = storeToRefs(store)

const socketItemCount = computed(() => {
  const items = filteredAndSortedSockets.value
  let count = 0
  for (const item of items) {
    if (item._type === 'socket') count++
  }
  return count
})

const isAllCollapsed = computed(() => {
  if (store.groupCount === 0) return false
  return collapsedGroups.value.size === store.groupCount
})

const toggleAll = () => {
  if (isAllCollapsed.value) {
    store.expandAll()
  } else {
    store.collapseAll()
  }
}

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
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  background-color: #1a1a2e;
  border-bottom: 1px solid #2d2d44;
  padding-right: 12px;
}

.header-row {
  display: grid;
  grid-template-columns: 60px 1fr 1fr 100px 140px 160px;
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

.search-row {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  background-color: #1a1a2e;
  border-bottom: 1px solid #2d2d44;
}

.search-icon {
  color: #606080;
  font-size: 16px;
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  background: #12122a;
  border: 1px solid #2d2d44;
  border-radius: 3px;
  padding: 4px 8px;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  color: #e0e0e0;
  outline: none;
}

.search-input:focus {
  border-color: #4a9eff;
}

.search-input::placeholder {
  color: #4a4a6a;
}

.search-error {
  font-family: 'Courier New', monospace;
  font-size: 10px;
  color: #ef4444;
  flex-shrink: 0;
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fold-all-btn {
  font-family: 'Courier New', monospace;
  font-size: 10px;
  padding: 2px 8px;
  background: #2d2d44;
  color: #8080a0;
  border: 1px solid #3d3d54;
  border-radius: 3px;
  cursor: pointer;
  white-space: nowrap;
}

.fold-all-btn:hover {
  background: #3d3d54;
  color: #c0c0e0;
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

.group-header .chevron {
  color: #4a9eff;
  font-size: 12px;
  width: 12px;
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
  cursor: pointer;
}

.group-header:hover {
  background: #252545;
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

.group-header .container-group-badge {
  font-size: 10px;
  color: #e0e0ff;
  background: #7c3aed;
  padding: 2px 6px;
  border-radius: 3px;
  font-weight: 600;
  margin-left: 8px;
}
</style>

<style>
/* Visible scrollbar for dark theme */
.vue-recycle-scroller {
  scrollbar-color: #3d3d54 #0f0f1a;
  scrollbar-width: thin;
}
.vue-recycle-scroller::-webkit-scrollbar {
  width: 8px;
}
.vue-recycle-scroller::-webkit-scrollbar-track {
  background: #0f0f1a;
}
.vue-recycle-scroller::-webkit-scrollbar-thumb {
  background: #3d3d54;
  border-radius: 4px;
}
.vue-recycle-scroller::-webkit-scrollbar-thumb:hover {
  background: #5a5a7a;
}
</style>

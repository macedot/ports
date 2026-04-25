<template>
  <div id="app">
    <header class="header">
      <div class="header-left">
        <h1 class="title">PORTS</h1>
      </div>
      <div class="header-right">
        <span v-if="updatedAt" class="timestamp">{{ formatTime(updatedAt) }}</span>
        <button class="refresh-btn" @click="refresh" :disabled="isLoading">
          <span v-if="isLoading && !error" class="spinner"></span>
          <span v-else>↻</span>
        </button>
      </div>
    </header>

    <div v-if="error" class="error-banner">
      {{ error }}
    </div>

    <FilterBar />
    <SocketTable />
  </div>
</template>

<script setup>
import { storeToRefs } from 'pinia'
import { useSocketsStore } from './stores/sockets'
import { usePolling } from './composables/usePolling'
import FilterBar from './components/FilterBar.vue'
import SocketTable from './components/SocketTable.vue'

const store = useSocketsStore()
const { error, isLoading, updatedAt } = storeToRefs(store)
const { refresh } = usePolling(store)

function formatTime(timestamp) {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  return date.toLocaleTimeString('en-US', { hour12: false })
}
</script>

<style scoped>
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background-color: #1a1a2e;
  border-bottom: 1px solid #2d2d44;
}

.header-left {
  display: flex;
  align-items: center;
}

.title {
  font-size: 20px;
  font-weight: 700;
  color: #e0e0ff;
  letter-spacing: 2px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.timestamp {
  font-family: 'Courier New', monospace;
  font-size: 12px;
  color: #606080;
}

.refresh-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: 1px solid #2d2d44;
  border-radius: 6px;
  background-color: #252540;
  color: #a0a0c0;
  font-size: 18px;
  cursor: pointer;
  transition: background-color 0.15s ease, color 0.15s ease;
}

.refresh-btn:hover:not(:disabled) {
  background-color: #3d3d6b;
  color: #e0e0ff;
}

.refresh-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid #606080;
  border-top-color: #a0a0c0;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.error-banner {
  padding: 10px 20px;
  background-color: #3d1a1a;
  border-bottom: 1px solid #6b2d2d;
  color: #ff8080;
  font-size: 13px;
}
</style>
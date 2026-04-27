<template>
  <div v-if="!needsAuth || isAuthenticated" id="app">
    <header class="header">
      <div class="header-left">
        <h1 class="title">PORTS</h1>
        <span v-if="serverVersion" class="version">v{{ serverVersion }}</span>
      </div>
      <div class="header-right">
        <span v-if="updatedAt" class="timestamp">{{ formatTime(updatedAt) }}</span>
        <button v-if="needsAuth" class="logout-btn" @click="handleLogout" title="Logout">🔒</button>
        <button class="refresh-btn" @click="refresh" :disabled="isLoading">
          <span v-if="isLoading && !error" class="spinner"></span>
          <span v-else>↻</span>
        </button>
      </div>
    </header>

    <div v-if="error" class="error-banner">
      {{ error }}
    </div>

    <FilterBar :is-frozen="isFrozen" @toggle-freeze="toggleFreeze" />
    <SocketTable />
  </div>

  <div v-if="needsAuth && !isAuthenticated" class="login-overlay">
    <div class="login-card">
      <h2 class="login-title">PORTS</h2>
      <form @submit.prevent="handleLogin(tokenInput)" class="login-form">
        <input
          ref="tokenInputRef"
          v-model="tokenInput"
          type="password"
          placeholder="Enter admin token"
          class="login-input"
          :disabled="authLoading"
          autocomplete="current-password"
        />
        <button type="submit" class="login-btn" :disabled="authLoading || !tokenInput">
          <span v-if="authLoading" class="spinner"></span>
          <span v-else>Authenticate</span>
        </button>
        <p v-if="authError" class="login-error">{{ authError }}</p>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, nextTick } from 'vue'
import { storeToRefs } from 'pinia'
import { useSocketsStore } from './stores/sockets'
import { usePolling } from './composables/usePolling'
import FilterBar from './components/FilterBar.vue'
import SocketTable from './components/SocketTable.vue'

const store = useSocketsStore()
const { error, isLoading, updatedAt, serverVersion } = storeToRefs(store)
const { refresh, isFrozen, toggleFreeze } = usePolling(store)

// Auth state
const authToken = ref(sessionStorage.getItem('admin_token') || '')
const isAuthenticated = ref(false)
const authError = ref('')
const authLoading = ref(false)
const needsAuth = ref(null)
const tokenInput = ref('')
const tokenInputRef = ref(null)

async function checkAuth() {
  const token = sessionStorage.getItem('admin_token')
  if (token) {
    try {
      const res = await fetch('/api/auth', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
      if (res.ok) {
        const data = await res.json()
        if (data.valid) {
          isAuthenticated.value = true
          needsAuth.value = false
          return
        }
      }
      sessionStorage.removeItem('admin_token')
      authToken.value = ''
      isAuthenticated.value = false
      needsAuth.value = true
    } catch {
      sessionStorage.removeItem('admin_token')
      authToken.value = ''
      isAuthenticated.value = false
      needsAuth.value = true
    }
  } else {
    try {
      const res = await fetch('/api/sockets')
      if (res.status === 401) {
        needsAuth.value = true
        isAuthenticated.value = false
      } else {
        needsAuth.value = false
        isAuthenticated.value = true
      }
    } catch {
      needsAuth.value = true
      isAuthenticated.value = false
    }
  }
}

async function handleLogin(token) {
  authLoading.value = true
  authError.value = ''
  try {
    const res = await fetch('/api/auth', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    if (res.ok) {
      const data = await res.json()
      if (data.valid) {
        sessionStorage.setItem('admin_token', token)
        authToken.value = token
        isAuthenticated.value = true
        needsAuth.value = false
        authError.value = ''
        tokenInput.value = ''
        return
      }
    }
    authError.value = 'Invalid token'
  } catch {
    authError.value = 'Invalid token'
  } finally {
    authLoading.value = false
  }
}

function handleLogout() {
  sessionStorage.removeItem('admin_token')
  authToken.value = ''
  isAuthenticated.value = false
  needsAuth.value = true
}

function formatTime(timestamp) {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  return date.toLocaleTimeString('en-US', { hour12: false })
}

onMounted(() => {
  checkAuth()
  window.addEventListener('auth:required', () => {
    isAuthenticated.value = false
    needsAuth.value = true
    sessionStorage.removeItem('admin_token')
  })
})

watch(() => needsAuth.value, (val) => {
  if (val === true) {
    nextTick(() => {
      tokenInputRef.value?.focus()
    })
  }
})
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
  gap: 12px;
}

.title {
  font-size: 20px;
  font-weight: 700;
  color: #e0e0ff;
  letter-spacing: 2px;
}

.version {
  font-family: 'Courier New', monospace;
  font-size: 10px;
  color: #4a4a6a;
  background: #252540;
  padding: 2px 6px;
  border-radius: 3px;
  letter-spacing: 0.5px;
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

.login-overlay {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: rgba(0, 0, 0, 0.7);
  z-index: 1000;
}

.login-card {
  background-color: #1a1a2e;
  border: 1px solid #2d2d44;
  border-radius: 12px;
  padding: 40px 32px;
  text-align: center;
  min-width: 320px;
}

.login-title {
  font-size: 24px;
  font-weight: 700;
  color: #e0e0ff;
  letter-spacing: 3px;
  margin: 0 0 28px 0;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.login-input {
  padding: 12px 16px;
  background-color: #252540;
  border: 1px solid #3d3d6b;
  border-radius: 6px;
  color: #e0e0ff;
  font-size: 14px;
  font-family: 'Courier New', monospace;
  outline: none;
  transition: border-color 0.15s ease;
}

.login-input:focus {
  border-color: #5d5d9b;
}

.login-input::placeholder {
  color: #606080;
  font-family: 'Courier New', monospace;
}

.login-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 12px 20px;
  background-color: #3d3d6b;
  border: none;
  border-radius: 6px;
  color: #e0e0ff;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.15s ease;
  min-height: 42px;
}

.login-btn:hover:not(:disabled) {
  background-color: #4d4d8b;
}

.login-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.login-error {
  color: #ff6060;
  font-size: 13px;
  margin: 0;
}

.logout-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 1px solid #2d2d44;
  border-radius: 6px;
  background-color: #252540;
  color: #a0a0c0;
  font-size: 14px;
  cursor: pointer;
  transition: background-color 0.15s ease, color 0.15s ease;
}

.logout-btn:hover {
  background-color: #3d3d6b;
  color: #e0e0ff;
}
</style>
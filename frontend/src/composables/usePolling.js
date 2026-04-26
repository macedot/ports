import { ref, onUnmounted } from 'vue'

/**
 * Polls the backend API every 5 seconds with tab visibility awareness.
 * @param {import('../stores/sockets').useSocketsStore} store - Pinia sockets store
 * @returns {{ refresh: () => void, isFrozen: import('vue').Ref<boolean>, toggleFreeze: () => void }}
 */
export function usePolling(store) {
  const INTERVAL_MS = 5000
  let intervalId = null
  let abortController = new AbortController()

  const isPolling = ref(false)
  const isFrozen = ref(false)

  function shouldSkip() {
    return store.isLoading
  }

  async function fetchSockets() {
    if (shouldSkip()) {
      return
    }

    abortController.abort()
    const controller = new AbortController()
    abortController = controller

    store.isLoading = true

    try {
      const token = sessionStorage.getItem('admin_token')
      const headers = {}
      if (token) {
        headers['Authorization'] = `Bearer ${token}`
      }
      const res = await fetch('/api/sockets', { signal: controller.signal, headers })
      if (res.status === 401) {
        sessionStorage.removeItem('admin_token')
        window.dispatchEvent(new CustomEvent('auth:required'))
        return
      }
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}: ${res.statusText}`)
      }
      const data = await res.json()
      store.setSockets(data.sockets, data.updated_at, data.docker_error)
      store.setError(null)
    } catch (err) {
      if (err.name === 'AbortError') {
        return
      }
      store.setError(err.message)
    } finally {
      store.isLoading = false
    }
  }

  function startPolling() {
    if (intervalId !== null) return
    intervalId = setInterval(fetchSockets, INTERVAL_MS)
    isPolling.value = true
  }

  function stopPolling() {
    if (intervalId !== null) {
      clearInterval(intervalId)
      intervalId = null
      isPolling.value = false
    }
  }

  function onVisibilityChange() {
    if (document.visibilityState === 'visible') {
      if (!isFrozen.value) {
        fetchSockets()
        startPolling()
      }
    } else {
      stopPolling()
    }
  }

  function toggleFreeze() {
    isFrozen.value = !isFrozen.value
    if (isFrozen.value) {
      stopPolling()
    } else {
      fetchSockets()
      startPolling()
    }
  }

  document.addEventListener('visibilitychange', onVisibilityChange)
  fetchSockets()
  startPolling()

  /**
   * Immediately fetch and reset the interval timer.
   */
  function refresh() {
    stopPolling()
    fetchSockets()
    startPolling()
  }

  onUnmounted(() => {
    stopPolling()
    abortController.abort()
    document.removeEventListener('visibilitychange', onVisibilityChange)
  })

  return { refresh, fetchSockets, isFrozen, toggleFreeze }
}
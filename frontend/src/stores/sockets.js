import { defineStore } from 'pinia'
import { shallowRef, ref, computed, triggerRef } from 'vue'

export const useSocketsStore = defineStore('sockets', () => {
  // State
  const sockets = shallowRef([])
  const protoFilter = ref('both')
  const ipVerFilter = ref('both')
  const sortKey = ref('local_port')
  const sortDir = ref('asc')
  const updatedAt = ref(null)
  const error = ref(null)
  const isLoading = ref(false)

  // Getters
  const filteredSockets = computed(() => {
    return sockets.value.filter((sock) => {
      // Proto filter
      if (protoFilter.value === 'tcp' && !sock.protocol.startsWith('TCP')) return false
      if (protoFilter.value === 'udp' && !sock.protocol.startsWith('UDP')) return false

      // IP version filter
      if (ipVerFilter.value === '4' && sock.protocol.endsWith('6')) return false
      if (ipVerFilter.value === '6' && !sock.protocol.endsWith('6')) return false

      return true
    })
  })

  const sortedSockets = computed(() => {
    const key = sortKey.value
    const dir = sortDir.value === 'asc' ? 1 : -1

    return [...filteredSockets.value].sort((a, b) => {
      const aVal = a[key]
      const bVal = b[key]

      // Numeric fields
      if (['local_port', 'remote_port'].includes(key)) {
        return (aVal - bVal) * dir
      }

      // String fields (case-insensitive)
      return String(aVal).localeCompare(String(bVal), undefined, { sensitivity: 'base' }) * dir
    })
  })

  const filteredAndSortedSockets = computed(() =>
    sortedSockets.value.map((s, i) => ({
      ...s,
      _key: `${s.protocol}-${s.local_addr}:${s.local_port}-${s.remote_addr}:${s.remote_port}-${i}`
    }))
  )

  // Actions
  function setSockets(data, timestamp) {
    sockets.value = data
    updatedAt.value = timestamp
    triggerRef(sockets)
  }

  function setProtoFilter(value) {
    if (['tcp', 'udp', 'both'].includes(value)) {
      protoFilter.value = value
    }
  }

  function setIPVerFilter(value) {
    if (['4', '6', 'both'].includes(value)) {
      ipVerFilter.value = value
    }
  }

  function toggleSort(key) {
    const sortableKeys = ['protocol', 'local_addr', 'local_port', 'remote_addr', 'remote_port', 'state', 'process']
    if (!sortableKeys.includes(key)) return

    if (sortKey.value === key) {
      sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
    } else {
      sortKey.value = key
      sortDir.value = 'asc'
    }
  }

  function setError(err) {
    error.value = err
  }

  function clearError() {
    error.value = null
  }

  return {
    // State
    sockets,
    protoFilter,
    ipVerFilter,
    sortKey,
    sortDir,
    updatedAt,
    error,
    isLoading,
    // Getters
    filteredSockets,
    sortedSockets,
    filteredAndSortedSockets,
    // Actions
    setSockets,
    setProtoFilter,
    setIPVerFilter,
    toggleSort,
    setError,
    clearError
  }
})

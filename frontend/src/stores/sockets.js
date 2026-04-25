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
  const collapsedGroups = ref(new Set())

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

      let cmp
      if (['local_port', 'remote_port'].includes(key)) {
        cmp = (aVal - bVal)
      } else {
        cmp = String(aVal).localeCompare(String(bVal), undefined, { sensitivity: 'base' })
      }

      // Secondary sort by local_port to keep groups contiguous
      if (cmp === 0 && key !== 'local_port') {
        return a.local_port - b.local_port
      }

      return cmp * dir
    })
  })

  const filteredAndSortedSockets = computed(() => {
    const result = []
    let currentGroup = null
    let groupIndex = 0

    // Build port counts in a single pass
    const portCounts = new Map()
    for (const s of sortedSockets.value) {
      portCounts.set(s.local_port, (portCounts.get(s.local_port) || 0) + 1)
    }

    for (let i = 0; i < sortedSockets.value.length; i++) {
      const s = sortedSockets.value[i]
      if (currentGroup !== s.local_port) {
        // Close previous group if exists
        if (currentGroup !== null) {
          groupIndex++
        }
        // Start new group - use pre-computed port count
        result.push({
          _type: 'group',
          port: s.local_port,
          count: portCounts.get(s.local_port),
          _key: `group-${s.local_port}-${groupIndex}`
        })
        currentGroup = s.local_port
      }
      // Only include socket item if group is not collapsed
      if (!collapsedGroups.value.has(s.local_port)) {
        result.push({
          ...s,
          _type: 'socket',
          _key: `${s.protocol}-${s.local_addr}:${s.local_port}-${s.remote_addr}:${s.remote_port}-${i}`
        })
      }
    }
    return result
  })

  const groupCount = computed(() => {
    const ports = new Set()
    for (const s of sortedSockets.value) {
      ports.add(s.local_port)
    }
    return ports.size
  })

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

  function toggleGroup(port) {
    if (collapsedGroups.value.has(port)) {
      collapsedGroups.value.delete(port)
    } else {
      collapsedGroups.value.add(port)
    }
    collapsedGroups.value = new Set(collapsedGroups.value)
  }

  function collapseAll() {
    const ports = new Set()
    for (const s of sortedSockets.value) {
      ports.add(s.local_port)
    }
    collapsedGroups.value = ports
  }

  function expandAll() {
    collapsedGroups.value = new Set()
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
    collapsedGroups,
    // Getters
    filteredSockets,
    sortedSockets,
    filteredAndSortedSockets,
    groupCount,
    // Actions
    setSockets,
    setProtoFilter,
    setIPVerFilter,
    toggleSort,
    setError,
    clearError,
    toggleGroup,
    collapseAll,
    expandAll
  }
})

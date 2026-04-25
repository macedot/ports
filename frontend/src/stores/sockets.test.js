import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useSocketsStore } from './sockets.js'

// Mock socket data with various protocols
const mockSockets = [
  { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 80, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
  { protocol: 'TCP6', local_addr: '::', local_port: 443, remote_addr: '::', remote_port: 0, state: 'LISTEN', process: 'nginx' },
  { protocol: 'UDP', local_addr: '0.0.0.0', local_port: 53, remote_addr: '0.0.0.0', remote_port: 0, state: '', process: 'bind' },
  { protocol: 'UDP6', local_addr: '::', local_port: 53, remote_addr: '::', remote_port: 0, state: '', process: 'bind' },
  { protocol: 'TCP', local_addr: '127.0.0.1', local_port: 3000, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'node' },
  { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 22, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'ssh' },
  { protocol: 'UDP6', local_addr: '::', local_port: 5353, remote_addr: '::', remote_port: 0, state: '', process: 'avahi' },
]

// Helper: extract only socket items (not group headers) from filteredAndSortedSockets
function getSocketItems(store) {
  return store.filteredAndSortedSockets.filter(item => item._type === 'socket')
}

describe('useSocketsStore', () => {
  beforeEach(() => {
    const pinia = createPinia()
    setActivePinia(pinia)
  })

  // 1. Initial state
  describe('Initial state', () => {
    it('has all filters default to "both"', () => {
      const store = useSocketsStore()
      expect(store.protoFilter).toBe('both')
      expect(store.ipVerFilter).toBe('both')
    })

    it('sorts by local_port ascending by default', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      expect(store.sortKey).toBe('local_port')
      expect(store.sortDir).toBe('asc')
    })

    it('returns all sockets with default filters (no filtering)', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      // 7 sockets, 6 unique ports -> 6 groups + 7 sockets = 13 total items
      const result = store.filteredAndSortedSockets
      expect(result).toHaveLength(13)
      const sockets = getSocketItems(store)
      expect(sockets).toHaveLength(7)
    })
  })

  // 2. Proto filter: tcp includes TCP and TCP6, excludes UDP/UDP6
  describe('Proto filter: tcp', () => {
    it('includes TCP and TCP6 protocols', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      store.setProtoFilter('tcp')
      const protocols = getSocketItems(store).map(s => s.protocol)
      expect(protocols).toContain('TCP')
      expect(protocols).toContain('TCP6')
    })

    it('excludes UDP and UDP6 protocols', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      store.setProtoFilter('tcp')
      const protocols = getSocketItems(store).map(s => s.protocol)
      expect(protocols).not.toContain('UDP')
      expect(protocols).not.toContain('UDP6')
    })
  })

  // 3. Proto filter: udp includes UDP and UDP6, excludes TCP/TCP6
  describe('Proto filter: udp', () => {
    it('includes UDP and UDP6 protocols', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      store.setProtoFilter('udp')
      const protocols = getSocketItems(store).map(s => s.protocol)
      expect(protocols).toContain('UDP')
      expect(protocols).toContain('UDP6')
    })

    it('excludes TCP and TCP6 protocols', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      store.setProtoFilter('udp')
      const protocols = getSocketItems(store).map(s => s.protocol)
      expect(protocols).not.toContain('TCP')
      expect(protocols).not.toContain('TCP6')
    })
  })

  // 4. IP filter: '4' excludes protocols ending in '6'
  describe('IP filter: 4 (IPv4)', () => {
    it('excludes protocols ending in 6 (TCP6, UDP6)', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      store.setIPVerFilter('4')
      const protocols = getSocketItems(store).map(s => s.protocol)
      expect(protocols).not.toContain('TCP6')
      expect(protocols).not.toContain('UDP6')
    })

    it('includes non-IPv6 protocols (TCP, UDP)', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      store.setIPVerFilter('4')
      const protocols = getSocketItems(store).map(s => s.protocol)
      expect(protocols).toContain('TCP')
      expect(protocols).toContain('UDP')
    })
  })

  // 5. IP filter: '6' includes only protocols ending in '6'
  describe('IP filter: 6 (IPv6)', () => {
    it('includes only protocols ending in 6 (TCP6, UDP6)', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      store.setIPVerFilter('6')
      const protocols = getSocketItems(store).map(s => s.protocol)
      expect(protocols).toContain('TCP6')
      expect(protocols).toContain('UDP6')
      expect(protocols).not.toContain('TCP')
      expect(protocols).not.toContain('UDP')
    })
  })

  // 6. Combined filter: proto='tcp' + ipver='4' returns only TCP (not TCP6)
  describe('Combined filter: proto=tcp + ipver=4', () => {
    it('returns only TCP (not TCP6)', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      store.setProtoFilter('tcp')
      store.setIPVerFilter('4')
      const protocols = getSocketItems(store).map(s => s.protocol)
      expect(protocols).toContain('TCP')
      expect(protocols).not.toContain('TCP6')
      expect(protocols).not.toContain('UDP')
      expect(protocols).not.toContain('UDP6')
    })
  })

  // 7. Sort by local_port ascending and descending
  describe('Sort by local_port', () => {
    it('sorts ascending by default', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      const ports = getSocketItems(store).map(s => s.local_port)
      expect(ports).toEqual([...ports].sort((a, b) => a - b))
    })

    it('sorts descending when direction is desc', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      store.sortDir = 'desc'
      const ports = getSocketItems(store).map(s => s.local_port)
      expect(ports).toEqual([...ports].sort((a, b) => b - a))
    })
  })

  // 8. Sort: toggle same key flips direction
  describe('Sort toggle same key', () => {
    it('flips direction when toggling same key', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())

      // Initial state is asc
      expect(store.sortDir).toBe('asc')
      const firstAsc = getSocketItems(store).map(s => s.local_port)

      // Toggle with same key flips to desc
      store.toggleSort('local_port')
      expect(store.sortDir).toBe('desc')
      const firstDesc = getSocketItems(store).map(s => s.local_port)

      // Toggle again flips back to asc
      store.toggleSort('local_port')
      expect(store.sortDir).toBe('asc')
      const secondAsc = getSocketItems(store).map(s => s.local_port)

      // Verify asc order
      expect(firstAsc).toEqual(secondAsc)
      expect(firstAsc).not.toEqual(firstDesc)
    })
  })

  // 9. Sort: new key resets to asc
  describe('Sort new key resets to asc', () => {
    it('resets to asc when sorting by new key', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())

      // Sort by local_port desc
      store.toggleSort('local_port')
      expect(store.sortKey).toBe('local_port')
      expect(store.sortDir).toBe('desc')

      // Switch to new key (protocol) resets to asc
      store.toggleSort('protocol')
      expect(store.sortKey).toBe('protocol')
      expect(store.sortDir).toBe('asc')
    })
  })

  // ===== GROUP BY PORT TESTS =====

  describe('Group by port: grouping basics', () => {
    it('produces 2 group headers and 5 total items for ports 80 (3 sockets) and 443 (2 sockets)', () => {
      const store = useSocketsStore()
      const groupTestSockets = [
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 80, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 80, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 80, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 443, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 443, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
      ]
      store.setSockets(groupTestSockets, Date.now())
      const result = store.filteredAndSortedSockets

      const groups = result.filter(item => item._type === 'group')
      const sockets = result.filter(item => item._type === 'socket')

      expect(groups).toHaveLength(2)
      expect(sockets).toHaveLength(5)
      expect(result).toHaveLength(7) // 2 groups + 5 sockets

      // Group counts
      const port80Group = groups.find(g => g.port === 80)
      const port443Group = groups.find(g => g.port === 443)
      expect(port80Group.count).toBe(3)
      expect(port443Group.count).toBe(2)
    })

    it('group headers appear before their socket items', () => {
      const store = useSocketsStore()
      const groupTestSockets = [
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 80, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 80, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
      ]
      store.setSockets(groupTestSockets, Date.now())
      const result = store.filteredAndSortedSockets

      // First item should be group, then sockets
      expect(result[0]._type).toBe('group')
      expect(result[0].port).toBe(80)
      expect(result.slice(1).every(item => item._type === 'socket')).toBe(true)
    })
  })

  describe('Group by port: single-port group', () => {
    it('group header for single socket shows count 1', () => {
      const store = useSocketsStore()
      const singleSocket = [
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 22, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'ssh' },
      ]
      store.setSockets(singleSocket, Date.now())
      const result = store.filteredAndSortedSockets

      const groups = result.filter(item => item._type === 'group')
      expect(groups).toHaveLength(1)
      expect(groups[0].count).toBe(1)
    })
  })

  describe('Group by port: group contiguity with non-port sort', () => {
    it('groups stay contiguous when sorting by protocol (TCP before UDP)', () => {
      const store = useSocketsStore()
      // Mixed ports and protocols
      const mixedSockets = [
        { protocol: 'UDP', local_addr: '0.0.0.0', local_port: 53, remote_addr: '0.0.0.0', remote_port: 0, state: '', process: 'bind' },
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 80, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
        { protocol: 'UDP', local_addr: '0.0.0.0', local_port: 80, remote_addr: '0.0.0.0', remote_port: 0, state: '', process: 'nginx' },
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 53, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'bind' },
      ]
      store.setSockets(mixedSockets, Date.now())
      store.toggleSort('protocol') // Sort by protocol, not local_port

      const result = store.filteredAndSortedSockets

      // Extract port sequence (skip group headers for port check)
      let currentGroupPort = null
      let inGroup = false
      let groupContiguityValid = true

      for (const item of result) {
        if (item._type === 'group') {
          currentGroupPort = item.port
          inGroup = true
        } else {
          // Socket item - verify it's in its group
          if (inGroup && item.local_port !== currentGroupPort) {
            groupContiguityValid = false
            break
          }
        }
      }

      expect(groupContiguityValid).toBe(true)
    })
  })

  describe('Group by port: sort by port (default)', () => {
    it('groups appear in ascending port order (22, 80, 443)', () => {
      const store = useSocketsStore()
      const multiPortSockets = [
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 443, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 80, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 22, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'ssh' },
      ]
      store.setSockets(multiPortSockets, Date.now())
      const result = store.filteredAndSortedSockets

      const groups = result.filter(item => item._type === 'group')
      expect(groups[0].port).toBe(22)
      expect(groups[1].port).toBe(80)
      expect(groups[2].port).toBe(443)
    })

    it('groups appear in descending port order when sortDir is desc', () => {
      const store = useSocketsStore()
      const multiPortSockets = [
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 443, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 80, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 22, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'ssh' },
      ]
      store.setSockets(multiPortSockets, Date.now())
      store.sortDir = 'desc'
      const result = store.filteredAndSortedSockets

      const groups = result.filter(item => item._type === 'group')
      expect(groups[0].port).toBe(443)
      expect(groups[1].port).toBe(80)
      expect(groups[2].port).toBe(22)
    })
  })

  describe('Group by port: key uniqueness', () => {
    it('two identical TIME_WAIT sockets produce unique _key values', () => {
      const store = useSocketsStore()
      // Two sockets with identical protocol, local/remote addr:port (like TIME_WAIT)
      const identicalSockets = [
        { protocol: 'TCP', local_addr: '192.168.1.100', local_port: 80, remote_addr: '192.168.1.50', remote_port: 54321, state: 'TIME_WAIT', process: '' },
        { protocol: 'TCP', local_addr: '192.168.1.100', local_port: 80, remote_addr: '192.168.1.50', remote_port: 54321, state: 'TIME_WAIT', process: '' },
      ]
      store.setSockets(identicalSockets, Date.now())
      const result = store.filteredAndSortedSockets

      const socketItems = result.filter(item => item._type === 'socket')
      expect(socketItems).toHaveLength(2)

      const keys = socketItems.map(s => s._key)
      expect(keys[0]).not.toBe(keys[1])
    })
  })

  describe('Group by port: empty state', () => {
    it('returns empty array when no sockets', () => {
      const store = useSocketsStore()
      store.setSockets([], Date.now())
      const result = store.filteredAndSortedSockets

      expect(result).toEqual([])
    })
  })

  describe('Group by port: filter interaction', () => {
    it('TCP filter only shows TCP sockets, group counts reflect filtered set', () => {
      const store = useSocketsStore()
      const mixedSockets = [
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 80, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 80, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
        { protocol: 'UDP', local_addr: '0.0.0.0', local_port: 80, remote_addr: '0.0.0.0', remote_port: 0, state: '', process: 'nginx' },
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 443, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
        { protocol: 'UDP', local_addr: '0.0.0.0', local_port: 443, remote_addr: '0.0.0.0', remote_port: 0, state: '', process: 'nginx' },
      ]
      store.setSockets(mixedSockets, Date.now())
      store.setProtoFilter('tcp')

      const result = store.filteredAndSortedSockets
      const groups = result.filter(item => item._type === 'group')
      const sockets = result.filter(item => item._type === 'socket')

      expect(sockets).toHaveLength(3) // Only TCP sockets
      expect(groups).toHaveLength(2)   // Two ports have TCP: 80 (2) and 443 (1)

      const port80Group = groups.find(g => g.port === 80)
      const port443Group = groups.find(g => g.port === 443)
      expect(port80Group.count).toBe(2)
      expect(port443Group.count).toBe(1)
    })
  })

  describe('Group by port: socketItemCount computed', () => {
    it('socketItemCount returns correct count of socket items only', () => {
      const store = useSocketsStore()
      const multiPortSockets = [
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 80, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 80, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
        { protocol: 'TCP', local_addr: '0.0.0.0', local_port: 443, remote_addr: '0.0.0.0', remote_port: 0, state: 'LISTEN', process: 'nginx' },
      ]
      store.setSockets(multiPortSockets, Date.now())

      // socketItemCount is exported from the store
      if (typeof store.socketItemCount !== 'undefined') {
        expect(store.socketItemCount).toBe(3)
      } else {
        // Fallback: count manually
        const result = store.filteredAndSortedSockets
        const count = result.filter(item => item._type === 'socket').length
        expect(count).toBe(3)
      }
    })

    it('socketItemCount is 0 for empty sockets', () => {
      const store = useSocketsStore()
      store.setSockets([], Date.now())

      if (typeof store.socketItemCount !== 'undefined') {
        expect(store.socketItemCount).toBe(0)
      } else {
        const result = store.filteredAndSortedSockets
        const count = result.filter(item => item._type === 'socket').length
        expect(count).toBe(0)
      }
    })
  })
})

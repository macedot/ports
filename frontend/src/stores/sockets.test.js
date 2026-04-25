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
      expect(store.filteredAndSortedSockets).toHaveLength(7)
    })
  })

  // 2. Proto filter: tcp includes TCP and TCP6, excludes UDP/UDP6
  describe('Proto filter: tcp', () => {
    it('includes TCP and TCP6 protocols', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      store.setProtoFilter('tcp')
      const protocols = store.filteredAndSortedSockets.map(s => s.protocol)
      expect(protocols).toContain('TCP')
      expect(protocols).toContain('TCP6')
    })

    it('excludes UDP and UDP6 protocols', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      store.setProtoFilter('tcp')
      const protocols = store.filteredAndSortedSockets.map(s => s.protocol)
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
      const protocols = store.filteredAndSortedSockets.map(s => s.protocol)
      expect(protocols).toContain('UDP')
      expect(protocols).toContain('UDP6')
    })

    it('excludes TCP and TCP6 protocols', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      store.setProtoFilter('udp')
      const protocols = store.filteredAndSortedSockets.map(s => s.protocol)
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
      const protocols = store.filteredAndSortedSockets.map(s => s.protocol)
      expect(protocols).not.toContain('TCP6')
      expect(protocols).not.toContain('UDP6')
    })

    it('includes non-IPv6 protocols (TCP, UDP)', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      store.setIPVerFilter('4')
      const protocols = store.filteredAndSortedSockets.map(s => s.protocol)
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
      const protocols = store.filteredAndSortedSockets.map(s => s.protocol)
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
      const protocols = store.filteredAndSortedSockets.map(s => s.protocol)
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
      const ports = store.filteredAndSortedSockets.map(s => s.local_port)
      expect(ports).toEqual([...ports].sort((a, b) => a - b))
    })

    it('sorts descending when direction is desc', () => {
      const store = useSocketsStore()
      store.setSockets(mockSockets, Date.now())
      store.sortDir = 'desc'
      const ports = store.filteredAndSortedSockets.map(s => s.local_port)
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
      const firstAsc = store.filteredAndSortedSockets.map(s => s.local_port)

      // Toggle with same key flips to desc
      store.toggleSort('local_port')
      expect(store.sortDir).toBe('desc')
      const firstDesc = store.filteredAndSortedSockets.map(s => s.local_port)

      // Toggle again flips back to asc
      store.toggleSort('local_port')
      expect(store.sortDir).toBe('asc')
      const secondAsc = store.filteredAndSortedSockets.map(s => s.local_port)

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
})

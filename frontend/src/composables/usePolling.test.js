// @vitest-environment jsdom
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

const createMockStore = () => ({
  isLoading: false,
  setSockets: vi.fn(),
  setError: vi.fn(),
})

function setupFetchMock(response, status = 200) {
  global.fetch = vi.fn(() =>
    Promise.resolve({
      status,
      ok: status >= 200 && status < 300,
      json: () => Promise.resolve(response),
    })
  )
}

describe('usePolling', () => {
  let mockStore
  let usePolling

  beforeEach(async () => {
    mockStore = createMockStore()
    sessionStorage.clear()
    vi.resetModules()
    ;({ usePolling } = await import('./usePolling.js'))
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('fetchSockets with token in sessionStorage', () => {
    it('includes Authorization header when token in sessionStorage', async () => {
      sessionStorage.setItem('admin_token', 'test-token-123')
      setupFetchMock({ sockets: [], updated_at: '2026-04-26T00:00:00Z' })

      const { fetchSockets } = usePolling(mockStore)
      await fetchSockets()

      expect(global.fetch).toHaveBeenCalledWith(
        '/api/sockets',
        expect.objectContaining({
          headers: {
            Authorization: 'Bearer test-token-123',
          },
        })
      )
    })
  })

  describe('fetchSockets without token in sessionStorage', () => {
    it('skips Authorization header when no token in sessionStorage', async () => {
      setupFetchMock({ sockets: [], updated_at: '2026-04-26T00:00:00Z' })

      const { fetchSockets } = usePolling(mockStore)
      await fetchSockets()

      expect(global.fetch).toHaveBeenCalledWith(
        '/api/sockets',
        expect.objectContaining({
          headers: {},
        })
      )
    })
  })

  describe('fetchSockets on 401 response', () => {
    it('clears sessionStorage and dispatches auth:required event on 401', async () => {
      sessionStorage.setItem('admin_token', 'expired-token')
      setupFetchMock({ error: 'unauthorized' }, 401)

      const dispatchSpy = vi.spyOn(window, 'dispatchEvent')
      const { fetchSockets } = usePolling(mockStore)
      await fetchSockets()

      expect(sessionStorage.getItem('admin_token')).toBeNull()
      expect(dispatchSpy).toHaveBeenCalled()
      const dispatchedEvent = dispatchSpy.mock.calls[0][0]
      expect(dispatchedEvent.type).toBe('auth:required')
    })
  })
})

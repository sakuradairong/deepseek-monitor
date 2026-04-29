import axios from 'axios'

const API_BASE = import.meta.env.VITE_API_BASE || '/api/v1'

const api = axios.create({
  baseURL: API_BASE,
  timeout: 15000,
})

// Request interceptor: attach auth token
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor: handle 401
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      window.location.hash = '#/login'
    }
    return Promise.reject(error)
  }
)

// ===== Auth =====
export function login(username, password) {
  return api.post('/auth/login', { username, password })
}

export function register(username, password) {
  return api.post('/auth/register', { username, password })
}

export function fetchMe() {
  return api.get('/auth/me')
}

// ===== Stats =====
export function fetchOverview() {
  return api.get('/stats/overview')
}

export function fetchBalanceHistory(days = 30) {
  return api.get('/stats/balance/history', { params: { days } })
}

export function fetchUsageTrend(days = 7) {
  return api.get('/stats/usage/trend', { params: { days } })
}

export function fetchUsageSummary(days = 30, model = '') {
  return api.get('/stats/usage/summary', { params: { days, model } })
}

export function fetchModelDistribution(days = 30) {
  return api.get('/stats/usage/models', { params: { days } })
}

export function fetchRateLimit() {
  return api.get('/stats/ratelimit')
}

export function fetchRecentErrors(limit = 20) {
  return api.get('/stats/errors', { params: { limit } })
}

// ===== Settings =====
export function fetchSettings() {
  return api.get('/settings')
}

export function updateSettings(data) {
  return api.put('/settings', data)
}

// ===== API Keys =====
export function fetchKeys() {
  return api.get('/keys')
}

export function createKey(data) {
  return api.post('/keys', data)
}

export function updateKey(id, data) {
  return api.put(`/keys/${id}`, data)
}

export function deleteKey(id) {
  return api.delete(`/keys/${id}`)
}

export function testKey(id) {
  return api.post(`/keys/${id}/test`)
}

export function fetchKeyNames() {
  return api.get('/keys/names')
}

// ===== Proxy / Realtime =====
export function fetchRealtimeMetrics() {
  return api.get('/proxy/realtime')
}

export function fetchProxyLogs(offset = 0, limit = 50, params = {}) {
  return api.get('/proxy/logs', { params: { offset, limit, ...params } })
}

export default api

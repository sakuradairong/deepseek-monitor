import { reactive, computed } from 'vue'
import { fetchMe } from '../api/index.js'

const state = reactive({
  token: localStorage.getItem('token') || '',
  user: JSON.parse(localStorage.getItem('user') || 'null'),
})

export function useAuth() {
  const isLoggedIn = computed(() => !!state.token && !!state.user)
  const username = computed(() => state.user?.username || '')
  const role = computed(() => state.user?.role || '')

  function setAuth(token, user) {
    state.token = token
    state.user = user
    localStorage.setItem('token', token)
    localStorage.setItem('user', JSON.stringify(user))
  }

  function logout() {
    state.token = ''
    state.user = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  async function verifySession() {
    if (!state.token) return false
    try {
      const { data } = await fetchMe()
      state.user = data
      localStorage.setItem('user', JSON.stringify(data))
      return true
    } catch {
      logout()
      return false
    }
  }

  return { state, isLoggedIn, username, role, setAuth, logout, verifySession }
}

import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { authApi } from '@/api'
import { clearToken, getToken, setToken } from '@/api/request'
import type { User } from '@/typings'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(getToken())
  const user = ref<User | null>(null)
  const loading = ref(false)

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 1)

  async function login(username: string, password: string) {
    loading.value = true
    try {
      const data = await authApi.login(username, password)
      token.value = data.token
      setToken(data.token)
      user.value = data.user
      return data.user
    } finally {
      loading.value = false
    }
  }

  async function fetchInfo() {
    if (!token.value) return null
    try {
      user.value = await authApi.info()
      return user.value
    } catch {
      reset()
      return null
    }
  }

  async function logout() {
    try {
      await authApi.logout()
    } catch {
      /* ignore */
    }
    reset()
  }

  function reset() {
    token.value = ''
    user.value = null
    clearToken()
  }

  return { token, user, loading, isLoggedIn, isAdmin, login, fetchInfo, logout, reset }
})

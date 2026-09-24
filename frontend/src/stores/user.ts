import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import * as authApi from '@/api/auth'
import * as usersApi from '@/api/users'
import { refreshSession } from '@/api/client'
import { setAccessToken } from '@/api/token'
import type { AuthResponse, LoginInput, RegisterInput, User } from '@/api/types'

export const useUserStore = defineStore('user', () => {
  const user = ref<User | null>(null)
  /** 是否已经尝试过恢复会话（防止每次路由跳转都打一次 /auth/refresh） */
  const initialized = ref(false)
  const loading = ref(false)

  const isLoggedIn = computed(() => user.value !== null)

  function setSession(data: AuthResponse): void {
    setAccessToken(data.access_token)
    user.value = data.user
  }

  function clear(): void {
    setAccessToken(null)
    user.value = null
  }

  async function login(input: LoginInput): Promise<void> {
    loading.value = true
    try {
      setSession(await authApi.login(input))
    } finally {
      loading.value = false
    }
  }

  async function register(input: RegisterInput): Promise<void> {
    loading.value = true
    try {
      setSession(await authApi.register(input))
    } finally {
      loading.value = false
    }
  }

  /**
   * 用 httpOnly Cookie 里的 refresh_token 恢复会话。
   *
   * 页面刷新后 access_token 就丢了（它只在内存里），
   * 靠这个函数把会话找回来，用户无感。
   */
  async function restore(): Promise<boolean> {
    try {
      setSession(await refreshSession())
      return true
    } catch {
      clear()
      return false
    } finally {
      initialized.value = true
    }
  }

  /** 首次进入应用时恢复一次会话 */
  async function ensureInitialized(): Promise<void> {
    if (!initialized.value) {
      await restore()
    }
  }

  async function fetchMe(): Promise<void> {
    user.value = await usersApi.fetchMe()
  }

  async function logout(): Promise<void> {
    try {
      await authApi.logout()
    } finally {
      // 无论后端是否成功，本地状态都要清干净
      clear()
    }
  }

  return {
    user,
    initialized,
    loading,
    isLoggedIn,
    login,
    register,
    restore,
    ensureInitialized,
    fetchMe,
    logout,
    clear,
  }
})

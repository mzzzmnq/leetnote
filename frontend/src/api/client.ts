import axios, {
  type AxiosError,
  type AxiosInstance,
  type InternalAxiosRequestConfig,
} from 'axios'
import { getAccessToken, setAccessToken } from './token'
import type { ApiErrorBody, AuthResponse } from './types'

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1'

// ---------------------------------------------------------------
// 统一错误类型
//
// 把 axios 的各种失败形态（HTTP 错误 / 超时 / 断网）收敛成一种，
// 上层只需要 catch ApiError，不用再判断 error.response 存不存在。
// ---------------------------------------------------------------
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly details?: Record<string, string>

  constructor(status: number, code: string, message: string, details?: Record<string, string>) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = details
  }
}

function toApiError(error: AxiosError<ApiErrorBody>): ApiError {
  const status = error.response?.status ?? 0
  const body = error.response?.data

  if (body?.error) {
    return new ApiError(status, body.error.code, body.error.message, body.error.details)
  }
  if (error.code === 'ECONNABORTED') {
    return new ApiError(status, 'TIMEOUT', '请求超时，请稍后重试')
  }
  if (!error.response) {
    return new ApiError(0, 'NETWORK_ERROR', '无法连接服务器，请确认后端已启动（localhost:8080）')
  }
  return new ApiError(status, 'UNKNOWN', error.message)
}

// ---------------------------------------------------------------
// 两个 axios 实例
//
//  - client    : 业务请求用，带拦截器
//  - bareClient: 只给「刷新 token」用，不带拦截器
//
// 为什么必须分开：刷新接口本身返回 401 时，如果它也走响应拦截器，
// 拦截器会再次尝试刷新 → 无限递归。用裸实例从根上避免。
// ---------------------------------------------------------------
export const client: AxiosInstance = axios.create({
  baseURL: BASE_URL,
  timeout: 15_000,
  // 必须开启：refresh_token 在 httpOnly Cookie 里，跨端口请求要带上它
  withCredentials: true,
})

const bareClient: AxiosInstance = axios.create({
  baseURL: BASE_URL,
  timeout: 15_000,
  withCredentials: true,
})

// ---------------------------------------------------------------
// 会话刷新（单飞 / single-flight）
//
// 页面加载时可能有多个请求同时 401。如果每个都去刷新，会打出一堆
// 刷新请求——而 refresh token 是【轮换】的，后发的请求拿到的
// 已经是失效 token，反而把自己踢下线。
// 所以用同一个 Promise 让它们共享一次刷新。
// ---------------------------------------------------------------
let refreshPromise: Promise<AuthResponse> | null = null

export function refreshSession(): Promise<AuthResponse> {
  if (!refreshPromise) {
    refreshPromise = bareClient
      .post<AuthResponse>('/auth/refresh')
      .then((res) => {
        setAccessToken(res.data.access_token)
        return res.data
      })
      .finally(() => {
        refreshPromise = null
      })
  }
  return refreshPromise
}

// 刷新失败时的回调，由 main.ts 注入（跳登录页）
let onUnauthorized: (() => void) | null = null

export function setUnauthorizedHandler(handler: () => void): void {
  onUnauthorized = handler
}

// ---------------------------------------------------------------
// 请求拦截：注入 access token
// ---------------------------------------------------------------
client.interceptors.request.use((config) => {
  const token = getAccessToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// ---------------------------------------------------------------
// 响应拦截：401 时自动刷新并重放原请求
// ---------------------------------------------------------------
interface RetryableConfig extends InternalAxiosRequestConfig {
  _retry?: boolean
}

client.interceptors.response.use(
  (response) => response,

  async (error: AxiosError<ApiErrorBody>) => {
    const original = error.config as RetryableConfig | undefined

    const shouldRetry =
      error.response?.status === 401 &&
      original !== undefined &&
      original._retry !== true

    if (shouldRetry) {
      original._retry = true
      try {
        const session = await refreshSession()
        original.headers.Authorization = `Bearer ${session.access_token}`
        // 重放原请求：用户完全无感
        return await client.request(original)
      } catch {
        setAccessToken(null)
        onUnauthorized?.()
        return Promise.reject(toApiError(error))
      }
    }

    return Promise.reject(toApiError(error))
  },
)

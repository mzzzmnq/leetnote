// 与后端 DTO 一一对应的类型定义。
//
// 手动维护（而不是从 OpenAPI 自动生成）是当前阶段的选择：
// 类型不多，手动写更直观。等接口数量上来后，可以改用
// openapi-typescript 从 /openapi.json 自动生成，避免两边不同步。

export interface User {
  id: number
  username: string
  email: string
  avatar_url: string | null
  bio: string | null
  created_at: string
}

export interface AuthResponse {
  access_token: string
  token_type: string
  expires_in: number
  user: User
}

export interface RegisterInput {
  username: string
  email: string
  password: string
}

export interface LoginInput {
  /** 用户名或邮箱 */
  login: string
  password: string
}

export interface UpdateProfileInput {
  avatar_url?: string | null
  bio?: string | null
}

export interface ChangePasswordInput {
  old_password: string
  new_password: string
}

export interface GitHubAuthorizeResponse {
  authorize_url: string
}

export interface OAuthAccount {
  provider: string
  provider_login: string
  avatar_url: string | null
  created_at: string
}

/** 后端统一错误体 */
export interface ApiErrorBody {
  error: {
    code: string
    message: string
    /** 字段级校验错误，如 { "password": "长度不能少于 8" } */
    details?: Record<string, string>
  }
}

/** 后端统一分页体 */
export interface PageData<T> {
  items: T[]
  total: number
  page: number
  size: number
  pages: number
}

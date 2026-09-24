import { client } from './client'
import type {
  AuthResponse,
  GitHubAuthorizeResponse,
  LoginInput,
  RegisterInput,
} from './types'

export async function register(input: RegisterInput): Promise<AuthResponse> {
  const { data } = await client.post<AuthResponse>('/auth/register', input)
  return data
}

export async function login(input: LoginInput): Promise<AuthResponse> {
  const { data } = await client.post<AuthResponse>('/auth/login', input)
  return data
}

export async function logout(): Promise<void> {
  await client.post('/auth/logout')
}

/** 取 GitHub 授权地址。state 由服务端生成并写入 httpOnly Cookie。 */
export async function fetchGitHubAuthorizeURL(redirect?: string): Promise<string> {
  const { data } = await client.get<GitHubAuthorizeResponse>('/auth/github/authorize', {
    params: redirect ? { redirect } : undefined,
  })
  return data.authorize_url
}

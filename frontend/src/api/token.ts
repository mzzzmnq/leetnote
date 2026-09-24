// access_token 只存在【内存】里，不写 localStorage。
//
// 为什么不写 localStorage：任何 XSS 都能读到它。放在模块级变量里，
// 刷新页面就丢——但这没关系，refresh_token 在 httpOnly Cookie 里，
// 重新加载后调一次 /auth/refresh 就能拿回新的 access_token。

let accessToken: string | null = null

export function getAccessToken(): string | null {
  return accessToken
}

export function setAccessToken(token: string | null): void {
  accessToken = token
}

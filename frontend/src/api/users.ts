import { client } from './client'
import type { ChangePasswordInput, OAuthAccount, UpdateProfileInput, User } from './types'

export async function fetchMe(): Promise<User> {
  const { data } = await client.get<User>('/users/me')
  return data
}

export async function updateProfile(input: UpdateProfileInput): Promise<User> {
  const { data } = await client.patch<User>('/users/me', input)
  return data
}

export async function changePassword(input: ChangePasswordInput): Promise<void> {
  await client.post('/users/me/password', input)
}

export async function fetchOAuthAccounts(): Promise<OAuthAccount[]> {
  const { data } = await client.get<OAuthAccount[]>('/users/me/oauth-accounts')
  return data
}

export async function unlinkGitHub(): Promise<void> {
  await client.delete('/users/me/oauth-accounts/github')
}

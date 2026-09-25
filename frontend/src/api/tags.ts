import { client } from './client'
import type { Tag, TagInput, TagKind } from './types'
import { toParams } from '@/utils/query'

/** 标签总量很小，后端一次性返回全部，不分页 */
export async function listTags(kind?: TagKind): Promise<Tag[]> {
  const { data } = await client.get<Tag[]>('/tags', { params: toParams({ kind }) })
  return data
}

export async function createTag(input: TagInput): Promise<Tag> {
  const { data } = await client.post<Tag>('/tags', input)
  return data
}

export async function deleteTag(id: number): Promise<void> {
  await client.delete(`/tags/${id}`)
}

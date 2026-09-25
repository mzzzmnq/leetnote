import { client } from './client'
import type { SearchResponse } from './types'
import { toParams } from '@/utils/query'

export interface SearchParams {
  q: string
  /** 不传则笔记和题目都搜 */
  type?: 'note' | 'problem'
  limit?: number
}

export async function search(params: SearchParams): Promise<SearchResponse> {
  const { data } = await client.get<SearchResponse>('/search', {
    params: toParams(params),
  })
  return data
}

import { client } from './client'
import type { StatsOverview, TrendResponse } from './types'
import { toParams } from '@/utils/query'

export async function fetchOverview(): Promise<StatsOverview> {
  const { data } = await client.get<StatsOverview>('/stats/overview')
  return data
}

export async function fetchTrend(days = 30): Promise<TrendResponse> {
  const { data } = await client.get<TrendResponse>('/stats/trend', {
    params: toParams({ days }),
  })
  return data
}

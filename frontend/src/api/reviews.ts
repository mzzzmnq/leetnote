import { client } from './client'
import type { ReviewCard, ReviewStats } from './types'
import { toParams } from '@/utils/query'

/** 取当前到期待复习的卡片 */
export async function fetchDueCards(limit = 20): Promise<ReviewCard[]> {
  const { data } = await client.get<ReviewCard[]>('/reviews/due', {
    params: toParams({ limit }),
  })
  return data
}

/**
 * 提交一次自评，返回按 SM-2 更新后的卡片。
 *
 * rating 取值 0-5：
 *   0 = 完全没想起来　1 = 想起来了但很吃力　2 = 想起来但有错误
 *   3 = 勉强答对　　　4 = 答对　　　　　　　5 = 轻松答对
 * rating < 3 会被判为「没答上来」，间隔重置为 1 天。
 */
export async function submitReview(cardId: number, rating: number): Promise<ReviewCard> {
  const { data } = await client.post<ReviewCard>(`/reviews/${cardId}/submit`, { rating })
  return data
}

export async function fetchReviewStats(): Promise<ReviewStats> {
  const { data } = await client.get<ReviewStats>('/reviews/stats')
  return data
}

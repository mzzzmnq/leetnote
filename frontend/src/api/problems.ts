import { client } from './client'
import type { PageData, Problem, ProblemInput, ProblemQuery } from './types'
import { toParams } from '@/utils/query'

export async function listProblems(query: ProblemQuery = {}): Promise<PageData<Problem>> {
  const { data } = await client.get<PageData<Problem>>('/problems', {
    params: toParams(query),
  })
  return data
}

export async function getProblem(id: number): Promise<Problem> {
  const { data } = await client.get<Problem>(`/problems/${id}`)
  return data
}

export async function createProblem(input: ProblemInput): Promise<Problem> {
  const { data } = await client.post<Problem>('/problems', input)
  return data
}

export async function updateProblem(id: number, input: ProblemInput): Promise<Problem> {
  const { data } = await client.put<Problem>(`/problems/${id}`, input)
  return data
}

export async function deleteProblem(id: number): Promise<void> {
  await client.delete(`/problems/${id}`)
}

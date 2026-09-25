import { client } from './client'
import type {
  Note,
  NoteInput,
  NoteListItem,
  NoteQuery,
  PageData,
  Solution,
  SolutionInput,
} from './types'
import { toParams } from '@/utils/query'

export async function listNotes(query: NoteQuery = {}): Promise<PageData<NoteListItem>> {
  const { data } = await client.get<PageData<NoteListItem>>('/notes', {
    params: toParams(query),
  })
  return data
}

export async function getNote(id: number): Promise<Note> {
  const { data } = await client.get<Note>(`/notes/${id}`)
  return data
}

export async function createNote(input: NoteInput): Promise<Note> {
  const { data } = await client.post<Note>('/notes', input)
  return data
}

/** 全量更新：解法与标签也在同一次请求里提交（后端用事务保证原子性） */
export async function updateNote(id: number, input: NoteInput): Promise<Note> {
  const { data } = await client.put<Note>(`/notes/${id}`, input)
  return data
}

export async function deleteNote(id: number): Promise<void> {
  await client.delete(`/notes/${id}`)
}

/** 切换收藏，返回更新后的列表项 */
export async function toggleStar(id: number): Promise<NoteListItem> {
  const { data } = await client.post<NoteListItem>(`/notes/${id}/star`)
  return data
}

export async function listSolutions(noteId: number): Promise<Solution[]> {
  const { data } = await client.get<Solution[]>(`/notes/${noteId}/solutions`)
  return data
}

export async function createSolution(noteId: number, input: SolutionInput): Promise<Solution> {
  const { data } = await client.post<Solution>(`/notes/${noteId}/solutions`, input)
  return data
}

export async function updateSolution(id: number, input: SolutionInput): Promise<Solution> {
  const { data } = await client.put<Solution>(`/solutions/${id}`, input)
  return data
}

export async function deleteSolution(id: number): Promise<void> {
  await client.delete(`/solutions/${id}`)
}

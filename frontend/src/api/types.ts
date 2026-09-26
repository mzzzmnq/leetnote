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

// ---------------------------------------------------------------
// 题目 / 标签 / 笔记 / 解法
//
// 字段名与后端 DTO 一一对应（snake_case）。
// ---------------------------------------------------------------

export type Difficulty = 'Easy' | 'Medium' | 'Hard'
export type NoteStatus = 'draft' | 'published'
export type TagKind = 'algorithm' | 'data_structure' | 'topic'

export interface Problem {
  id: number
  leetcode_id: number | null
  title: string
  title_slug: string
  difficulty: Difficulty
  url: string | null
  created_at: string
  /** 所属专题/知识点（题单导入时写入） */
  tags: Tag[]
}

/** 嵌套在笔记里的精简题目信息 */
export interface ProblemBrief {
  id: number
  leetcode_id: number | null
  title: string
  difficulty: Difficulty
}

export interface ProblemInput {
  leetcode_id?: number | null
  title: string
  title_slug: string
  difficulty: Difficulty
  url?: string | null
}

export interface Tag {
  id: number
  name: string
  slug: string
  kind: TagKind
  created_at: string
}

export interface TagInput {
  name: string
  slug: string
  kind: TagKind
}

export interface Solution {
  id: number
  title: string
  language: string
  code: string
  time_complexity: string | null
  space_complexity: string | null
  sort_order: number
  created_at: string
}

export interface SolutionInput {
  title: string
  language: string
  code: string
  time_complexity?: string | null
  space_complexity?: string | null
}

/** 列表项：刻意不含 content_md */
export interface NoteListItem {
  id: number
  problem_id: number | null
  problem: ProblemBrief | null
  title: string
  summary: string | null
  status: NoteStatus
  is_starred: boolean
  tags: Tag[]
  solution_count: number
  created_at: string
  updated_at: string
}

/** 详情：含正文与全部解法 */
export interface Note extends NoteListItem {
  content_md: string
  view_count: number
  solutions: Solution[]
}

export interface NoteInput {
  problem_id?: number | null
  title: string
  content_md: string
  summary?: string | null
  status: NoteStatus
  is_starred?: boolean
  tag_ids?: number[]
  solutions?: SolutionInput[]
}

export interface NoteQuery {
  keyword?: string
  difficulty?: Difficulty | ''
  tag_id?: number | null
  status?: NoteStatus | ''
  starred?: boolean
  sort?: 'created' | 'updated' | 'created_asc' | 'title'
  page?: number
  size?: number
}

export interface ProblemQuery {
  keyword?: string
  difficulty?: Difficulty | ''
  tag_id?: number
  page?: number
  size?: number
}

// ---------------------------------------------------------------
// 统计与搜索
// ---------------------------------------------------------------

export interface DifficultyBreakdown {
  easy: number
  medium: number
  hard: number
}

export interface StatsOverview {
  total_notes: number
  total_problems: number
  total_solutions: number
  total_tags: number
  starred: number
  drafts: number
  /** 有记录的天数 */
  active_days: number
  /** 当前连续打卡天数 */
  current_streak: number
  /** 历史最长连续打卡 */
  longest_streak: number
  difficulty: DifficultyBreakdown
}

/** 趋势图上的一个点 */
export interface TrendPoint {
  /** YYYY-MM-DD */
  date: string
  count: number
}

export interface TrendResponse {
  days: number
  points: TrendPoint[]
}

/** 搜索结果：一次返回笔记与题目两类 */
export interface SearchResponse {
  keyword: string
  notes: NoteListItem[]
  problems: Problem[]
}


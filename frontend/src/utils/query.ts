/**
 * 把对象转成查询参数，自动丢弃 undefined / null / 空字符串。
 *
 * 为什么要丢弃空值：后端对 `difficulty` 这类字段有 `oneof=Easy Medium Hard`
 * 校验，传 `difficulty=` 会被判为非法值直接返回 422。
 * 统一在这里过滤，避免每个调用点都写一遍 if。
 */
export function toParams<T extends object>(input: T): Record<string, string> {
  const out: Record<string, string> = {}

  for (const [key, value] of Object.entries(input)) {
    if (value === undefined || value === null || value === '') {
      continue
    }
    out[key] = String(value)
  }

  return out
}

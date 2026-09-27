"""Prompt 模板。

集中放在一处便于调优与对比，也避免 prompt 散落在业务代码里 ——
prompt 是「会反复改的配置」，不是「一次写死的常量」。
"""

from __future__ import annotations

from collections.abc import Iterable

# 输入内容截断上限：防止一篇超长笔记把上下文撑爆
MAX_CONTENT_CHARS = 6000
MAX_CODE_CHARS = 2000
MAX_SOLUTIONS = 3

EXPLAIN_SYSTEM = """你是一位算法面试教练，擅长把解法讲清楚，而不是直接甩答案。

严格遵守：
1. 用中文回答，Markdown 格式。
2. 只输出下面四个二级标题，顺序不变，不要增加其它章节，不要写开头和结尾的客套话：
   ## 思路
   ## 关键点
   ## 复杂度
   ## 易错点
3. 全文控制在 400 字以内，不要重复题目原文。
4. 如果信息不足以判断，就基于现有信息合理推断，不要反问用户。
5. 不要输出代码块以外的多余格式；代码只在确有必要时给一小段。
"""


class SolutionBrief:
    """传给 prompt 的解法摘要（避免依赖 ORM 对象）。"""

    def __init__(
        self,
        title: str,
        language: str,
        code: str,
        time_complexity: str | None = None,
        space_complexity: str | None = None,
    ) -> None:
        self.title = title
        self.language = language
        self.code = code
        self.time_complexity = time_complexity
        self.space_complexity = space_complexity


def build_explain_prompt(
    *,
    title: str,
    problem_title: str | None,
    difficulty: str | None,
    content_md: str,
    solutions: Iterable[SolutionBrief],
) -> str:
    """拼出「点评这篇笔记」的 user prompt。"""
    parts: list[str] = []

    head = problem_title or title
    parts.append(f"# 题目\n{head}")
    if difficulty:
        parts.append(f"难度：{difficulty}")

    if content_md.strip():
        parts.append(f"\n# 我的笔记\n{content_md[:MAX_CONTENT_CHARS]}")

    codes: list[str] = []
    for sol in list(solutions)[:MAX_SOLUTIONS]:
        codes.append(
            f"### {sol.title}（{sol.language}）\n"
            f"复杂度：时间 {sol.time_complexity or '未标注'} / 空间 {sol.space_complexity or '未标注'}\n"
            f"```{sol.language}\n{sol.code[:MAX_CODE_CHARS]}\n```"
        )
    if codes:
        parts.append("\n# 我的解法\n" + "\n\n".join(codes))

    parts.append("\n请按系统要求的结构，点评并补充这篇笔记。")

    return "\n".join(parts)

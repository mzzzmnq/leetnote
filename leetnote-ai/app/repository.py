"""数据访问：notes 与 note_embeddings。

只碰这两张表 —— 业务数据的所有权仍在 Go 服务，这里只负责向量相关读写。
"""

from __future__ import annotations

import logging
from dataclasses import dataclass
from datetime import datetime

from app.db import get_pool

logger = logging.getLogger(__name__)


@dataclass(frozen=True, slots=True)
class NoteSource:
    """用于生成向量的原始内容。"""

    id: int
    user_id: int
    title: str
    content_md: str
    summary: str | None
    problem_title: str | None

    def to_document(self) -> str:
        """拼成一段用于向量化的文本。

        标题放最前面并重复一次：标题通常是整篇笔记信息密度最高的部分，
        重复能提高它在向量里的权重（本地哈希向量器对词频敏感）。
        """
        parts = [self.title, self.title]
        if self.problem_title:
            parts.append(self.problem_title)
        if self.summary:
            parts.append(self.summary)
        if self.content_md:
            parts.append(self.content_md)
        return "\n".join(parts)


@dataclass(frozen=True, slots=True)
class SimilarNote:
    note_id: int
    title: str
    summary: str | None
    is_starred: bool
    updated_at: datetime
    similarity: float


def _to_vector_literal(vec: list[float]) -> str:
    """转成 pgvector 能解析的字面量：[0.1,0.2,...]"""
    return "[" + ",".join(f"{v:.7f}" for v in vec) + "]"


async def fetch_note_source(note_id: int) -> NoteSource | None:
    pool = get_pool()

    async with pool.connection() as conn, conn.cursor() as cur:
        await cur.execute(
            """
            SELECT n.id, n.user_id, n.title, n.content_md, n.summary, p.title
            FROM notes n
            LEFT JOIN problems p ON p.id = n.problem_id
            WHERE n.id = %(note_id)s
            """,
            {"note_id": note_id},
        )
        row = await cur.fetchone()

    if row is None:
        return None

    return NoteSource(
        id=row[0],
        user_id=row[1],
        title=row[2] or "",
        content_md=row[3] or "",
        summary=row[4],
        problem_title=row[5],
    )


async def upsert_embedding(note_id: int, embedding: list[float], model: str) -> None:
    pool = get_pool()

    async with pool.connection() as conn, conn.cursor() as cur:
        await cur.execute(
            """
            INSERT INTO note_embeddings (note_id, embedding, model, updated_at)
            VALUES (%(note_id)s, %(embedding)s::vector, %(model)s, now())
            ON CONFLICT (note_id) DO UPDATE
            SET embedding  = EXCLUDED.embedding,
                model      = EXCLUDED.model,
                updated_at = now()
            """,
            {
                "note_id": note_id,
                "embedding": _to_vector_literal(embedding),
                "model": model,
            },
        )


async def find_similar(user_id: int, note_id: int, limit: int) -> list[SimilarNote]:
    """找出与指定笔记最相似的若干篇（同一用户内）。

    两个安全/正确性要点：

    1. **用户隔离**：CTE 里先确认这篇笔记属于该用户，
       传别人的 note_id 会得到空结果而不是别人的数据。

    2. **只比同模型的向量**：不同 embedding 模型产生的向量不在同一空间，
       混在一起算余弦相似度是没有意义的。所以限定 `e.model = t.model`。
    """
    pool = get_pool()

    async with pool.connection() as conn, conn.cursor() as cur:
        await cur.execute(
            """
            WITH target AS (
                SELECT e.embedding, e.model
                FROM note_embeddings e
                JOIN notes n ON n.id = e.note_id
                WHERE e.note_id = %(note_id)s AND n.user_id = %(user_id)s
            )
            SELECT
                n.id                                            AS note_id,
                n.title                                         AS title,
                n.summary                                       AS summary,
                n.is_starred                                    AS is_starred,
                n.updated_at                                    AS updated_at,
                1 - (e.embedding <=> t.embedding)               AS similarity
            FROM note_embeddings e
            JOIN notes n ON n.id = e.note_id
            CROSS JOIN target t
            WHERE n.user_id = %(user_id)s
              AND e.note_id <> %(note_id)s
              AND e.model = t.model
            ORDER BY e.embedding <=> t.embedding
            LIMIT %(limit)s
            """,
            {"user_id": user_id, "note_id": note_id, "limit": limit},
        )
        rows = await cur.fetchall()

    return [
        SimilarNote(
            note_id=r[0],
            title=r[1],
            summary=r[2],
            is_starred=r[3],
            updated_at=r[4],
            similarity=float(r[5]),
        )
        for r in rows
    ]


async def count_embeddings(user_id: int) -> tuple[int, int]:
    """返回 (已生成向量的笔记数, 该用户的笔记总数)。"""
    pool = get_pool()

    async with pool.connection() as conn, conn.cursor() as cur:
        await cur.execute(
            """
            SELECT
                count(e.note_id),
                count(n.id)
            FROM notes n
            LEFT JOIN note_embeddings e ON e.note_id = n.id
            WHERE n.user_id = %(user_id)s
            """,
            {"user_id": user_id},
        )
        row = await cur.fetchone()

    return (int(row[0]), int(row[1])) if row else (0, 0)

"""向量化与相似题检索。

所有接口都要求内部令牌（见 deps.verify_internal_token）。
"""

from __future__ import annotations

import logging

from fastapi import APIRouter, Depends, HTTPException, status

from app import repository
from app.api.deps import get_embedder, verify_internal_token
from app.embedding.base import Embedder
from app.schemas import (
    BatchEmbedRequest,
    BatchEmbedResponse,
    EmbedRequest,
    EmbedResponse,
    SimilarNoteOut,
    SimilarRequest,
    SimilarResponse,
)

logger = logging.getLogger(__name__)

router = APIRouter(tags=["ai"], dependencies=[Depends(verify_internal_token)])


async def _embed_and_store(note_id: int, embedder: Embedder) -> None:
    """读取笔记内容 → 生成向量 → 写库。"""
    source = await repository.fetch_note_source(note_id)
    if source is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"笔记 {note_id} 不存在",
        )

    vectors = await embedder.embed([source.to_document()])
    await repository.upsert_embedding(note_id, vectors[0], embedder.name)


@router.post("/embed", response_model=EmbedResponse)
async def embed_note(
    payload: EmbedRequest,
    embedder: Embedder = Depends(get_embedder),
) -> EmbedResponse:
    """为单篇笔记生成（或重建）向量。

    Go 服务在笔记创建/更新后调用。是【幂等】的：重复调用只会覆盖同一条记录。
    """
    await _embed_and_store(payload.note_id, embedder)

    return EmbedResponse(note_id=payload.note_id, model=embedder.name, dim=embedder.dim)


@router.post("/embed/batch", response_model=BatchEmbedResponse)
async def embed_notes_batch(
    payload: BatchEmbedRequest,
    embedder: Embedder = Depends(get_embedder),
) -> BatchEmbedResponse:
    """批量补向量。

    用途：导入历史笔记后一次性补齐，或者换了 embedding 模型后重建全部向量。
    单条失败不影响其余（记下失败数，不整体回滚）。
    """
    succeeded = 0
    failed = 0

    for note_id in payload.note_ids:
        try:
            await _embed_and_store(note_id, embedder)
            succeeded += 1
        except Exception as exc:  # noqa: BLE001 - 单条失败要能继续
            failed += 1
            logger.warning("笔记 %s 向量化失败: %s", note_id, exc)

    return BatchEmbedResponse(
        requested=len(payload.note_ids),
        succeeded=succeeded,
        failed=failed,
        model=embedder.name,
    )


@router.get("/embedding/status")
async def embedding_status(user_id: int) -> dict[str, int]:
    """查看某个用户的向量覆盖率（已生成 / 总笔记数）。"""
    embedded, total = await repository.count_embeddings(user_id)
    return {"embedded": embedded, "total": total}


@router.post("/similar", response_model=SimilarResponse)
async def find_similar(
    payload: SimilarRequest,
    embedder: Embedder = Depends(get_embedder),
) -> SimilarResponse:
    """找出与指定笔记最相似的其他笔记。"""
    limit = min(payload.limit, 20)

    notes = await repository.find_similar(
        user_id=payload.user_id,
        note_id=payload.note_id,
        limit=limit,
    )

    return SimilarResponse(
        note_id=payload.note_id,
        model=embedder.name,
        items=[
            SimilarNoteOut(
                note_id=n.note_id,
                title=n.title,
                summary=n.summary,
                is_starred=n.is_starred,
                # 保留 4 位小数就够了，避免前端显示一长串浮点数
                similarity=round(n.similarity, 4),
            )
            for n in notes
        ],
    )

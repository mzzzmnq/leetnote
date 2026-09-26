"""健康检查。"""

from __future__ import annotations

from fastapi import APIRouter, Depends

from app.api.deps import get_embedder
from app.core.config import Settings, get_settings
from app.db import get_pool
from app.embedding.base import Embedder
from app.schemas import HealthResponse

router = APIRouter(tags=["health"])


@router.get("/health", response_model=HealthResponse)
async def health(
    embedder: Embedder = Depends(get_embedder),
    settings: Settings = Depends(get_settings),
) -> HealthResponse:
    """健康检查。

    刻意【不】校验内部令牌：否则监控系统没法探活。
    但也【不】返回任何业务数据，只报告自身状态。
    """
    db_status = "ok"
    status_value = "ok"

    try:
        pool = get_pool()
        async with pool.connection() as conn, conn.cursor() as cur:
            await cur.execute("SELECT 1")
    except Exception:  # noqa: BLE001 - 健康检查不关心具体错误类型
        db_status = "down"
        status_value = "degraded"

    return HealthResponse(
        status=status_value,
        service="leetnote-ai",
        embedding_model=embedder.name,
        embedding_mode="remote" if settings.use_remote_embedding else "local",
        embedding_dim=embedder.dim,
        database=db_status,
    )

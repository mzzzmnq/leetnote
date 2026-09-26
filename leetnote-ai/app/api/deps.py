"""FastAPI 依赖注入。"""

from __future__ import annotations

import secrets

from fastapi import Header, HTTPException, Request, status

from app.core.config import get_settings
from app.embedding.base import Embedder


def get_embedder(request: Request) -> Embedder:
    """从应用状态里取向量器（在 lifespan 里初始化）。"""
    embedder = getattr(request.app.state, "embedder", None)
    if embedder is None:  # pragma: no cover - 正常启动流程不会走到
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="向量器尚未初始化",
        )
    return embedder


async def verify_internal_token(
    x_internal_token: str = Header(default="", alias="X-Internal-Token"),
) -> None:
    """校验服务间共享密钥。

    本服务只应由 Go 服务调用（用户鉴权在 Go 侧完成）。没有这道校验的话，
    只要这个端口能从网络访问到，任何人都能用任意 user_id 读到别人的数据。

    用 secrets.compare_digest 而不是 == ：普通字符串比较会在第一个不同的
    字符处提前返回，理论上可以逐字节爆破出正确值。
    """
    expected = get_settings().internal_api_token
    if not expected or not secrets.compare_digest(x_internal_token, expected):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="内部调用令牌无效",
        )

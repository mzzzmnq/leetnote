"""LeetNote AI 服务入口。

职责边界（与 Go 服务分工）：
  - Go 服务  ：业务数据（用户/题目/笔记/解法）的所有者，负责用户鉴权
  - 本服务   ：向量化 与 相似度检索，【不做用户认证】，只信任内部调用

因此本服务应当只监听内网，并用 X-Internal-Token 拦住直接访问。
"""

from __future__ import annotations

import logging
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

from fastapi import APIRouter, FastAPI

from app.api.v1 import health, similar
from app.core.config import get_settings
from app.core.logging import setup_logging
from app.db import close_pool, init_pool
from app.embedding import build_embedder

# 注意：Windows 上需要在事件循环创建【之前】切换到 SelectorEventLoop，
# 这一步写在 run.py 里（uvicorn 的顺序是「建循环 → 导入 app」，写在这里来不及）。
# 请用 `python run.py` 启动，不要直接 `uvicorn app.main:app`。

logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    settings = get_settings()
    setup_logging(settings.app_env)

    logger.info("正在启动 leetnote-ai (env=%s)", settings.app_env)

    # 数据库连不上就直接启动失败，不要带着坏状态对外服务
    await init_pool(settings.database_url)
    app.state.embedder = build_embedder(settings)

    logger.info("服务就绪，向量器=%s", app.state.embedder.name)
    try:
        yield
    finally:
        await close_pool()
        logger.info("服务已退出")


app = FastAPI(
    title="LeetNote AI Service",
    version="0.1.0",
    description="向量化与相似题检索（内部服务）",
    lifespan=lifespan,
)

# 探活接口放在根路径，且不校验令牌 —— 否则监控系统没法用
app.include_router(health.router)

api_v1 = APIRouter(prefix="/api/v1/ai")
api_v1.include_router(similar.router)
app.include_router(api_v1)

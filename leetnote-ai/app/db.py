"""数据库连接池。

用 psycopg3 的异步连接池。之所以不用 SQLAlchemy：
这个服务只碰两张表（notes / note_embeddings），手写 SQL 更直观，
也少一层 ORM 的心智负担。
"""

from __future__ import annotations

import logging

from psycopg_pool import AsyncConnectionPool

logger = logging.getLogger(__name__)

_pool: AsyncConnectionPool | None = None


async def init_pool(dsn: str) -> AsyncConnectionPool:
    """初始化连接池（幂等，重复调用返回同一个）。"""
    global _pool

    if _pool is not None:
        return _pool

    # open=False 后再显式 open()：psycopg_pool 3.2+ 不再推荐在构造函数里建立连接，
    # 因为那样无法在异步环境里正确 await。
    pool = AsyncConnectionPool(dsn, min_size=1, max_size=5, open=False)
    await pool.open()
    await pool.wait()

    _pool = pool
    logger.info("数据库连接池就绪 (min=1, max=5)")
    return pool


async def close_pool() -> None:
    global _pool
    if _pool is not None:
        await _pool.close()
        _pool = None
        logger.info("数据库连接池已关闭")


def get_pool() -> AsyncConnectionPool:
    if _pool is None:
        raise RuntimeError("连接池尚未初始化，请先调用 init_pool()")
    return _pool

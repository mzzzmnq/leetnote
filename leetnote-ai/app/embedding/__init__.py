"""Embedding 工厂。

根据配置决定用远程模型还是本地兜底 —— 调用方完全不关心这个选择。
"""

from __future__ import annotations

import logging

from app.core.config import Settings
from app.embedding.base import Embedder
from app.embedding.local import LocalHashingEmbedder
from app.embedding.openai import OpenAIEmbedder

logger = logging.getLogger(__name__)


def build_embedder(settings: Settings) -> Embedder:
    if settings.use_remote_embedding:
        logger.info(
            "使用远程 embedding 模型: %s (dim=%d, base_url=%s)",
            settings.embedding_model,
            settings.embedding_dim,
            settings.llm_base_url,
        )
        return OpenAIEmbedder(
            base_url=settings.llm_base_url,
            api_key=settings.llm_api_key,
            model=settings.embedding_model,
            dim=settings.embedding_dim,
        )

    logger.warning(
        "未配置 LLM_API_KEY / EMBEDDING_MODEL，回退到本地哈希向量器。"
        "它只捕获字面相似度，不理解语义 —— 适合开发调试，不适合生产。"
    )
    return LocalHashingEmbedder(dim=settings.embedding_dim)


__all__ = ["Embedder", "LocalHashingEmbedder", "OpenAIEmbedder", "build_embedder"]

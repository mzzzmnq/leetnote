"""LLM 工厂。

根据配置决定是否启用对话模型，调用方不需要关心供应商差异。
"""

from __future__ import annotations

import logging

from app.core.config import Settings
from app.llm.client import ChatClient, ChatError, ChatResult

logger = logging.getLogger(__name__)


def build_chat_client(settings: Settings) -> ChatClient | None:
    if not settings.use_chat_model:
        logger.warning(
            "未配置 CHAT_API_KEY / CHAT_MODEL，LLM 相关功能（解法讲解、复习卡）不可用"
        )
        return None

    logger.info(
        "对话模型已启用: %s (base_url=%s, ua=%s)",
        settings.chat_model,
        settings.chat_base_url,
        settings.chat_user_agent or "(默认)",
    )
    return ChatClient(
        base_url=settings.chat_base_url,
        api_key=settings.chat_api_key,
        model=settings.chat_model,
        user_agent=settings.chat_user_agent,
        session_header=settings.chat_session_header,
    )


__all__ = ["ChatClient", "ChatError", "ChatResult", "build_chat_client"]

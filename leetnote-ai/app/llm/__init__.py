"""LLM 工厂。

根据配置决定是否启用对话模型，调用方不需要关心供应商差异。
"""

from __future__ import annotations

import logging

from app.core.config import Settings
from app.llm.client import ChatClient, ChatError, ChatResult
from app.llm.discovery import discover_local_api_key

logger = logging.getLogger(__name__)


def build_chat_client(settings: Settings) -> ChatClient | None:
    """按「环境变量 → 本机自动发现 → 不可用」的顺序取 key，构建对话模型客户端。

    【为什么自动发现】LeetNote 的 AI 讲解由 OpenCode Go 订阅驱动，而 opencode
    已把 key 存在本机凭据库里。对个人本机使用，再让人手填一遍 .env 是多余的。
    显式的 CHAT_API_KEY 仍然优先，方便在多机/多账号场景覆盖。
    """
    api_key = settings.chat_api_key.strip()
    source = "环境变量 CHAT_API_KEY"
    if not api_key:
        discovered = discover_local_api_key()
        if discovered:
            api_key = discovered
            source = "本机 opencode 凭据（自动发现）"

    if not api_key:
        logger.warning(
            "未配置对话模型：CHAT_API_KEY 为空且本机未发现 OpenCode Go key，"
            "LLM 相关功能（解法讲解）不可用"
        )
        return None

    logger.info(
        "对话模型已启用: %s (base_url=%s, key 来源=%s)",
        settings.effective_chat_model,
        settings.effective_chat_base_url,
        source,
    )
    return ChatClient(
        base_url=settings.effective_chat_base_url,
        api_key=api_key,
        model=settings.effective_chat_model,
        user_agent=settings.chat_user_agent,
        session_header=settings.chat_session_header,
    )


__all__ = ["ChatClient", "ChatError", "ChatResult", "build_chat_client"]

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
    """按「环境变量 → （仅 OpenCode Go）本机自动发现 → 不可用」取 key，构建对话模型客户端。

    默认端点是智谱 GLM-4.5-Flash（免费）。自动发现只对 OpenCode Go 生效 ——
    否则会把 OpenCode Go 的 key 发到智谱等平台上，既不安全也没意义。
    """
    api_key = settings.chat_api_key.strip()
    source = "环境变量 CHAT_API_KEY"

    if not api_key and settings.uses_opencode_go:
        discovered = discover_local_api_key()
        if discovered:
            api_key = discovered
            source = "本机 opencode 凭据（自动发现）"

    if not api_key:
        logger.warning(
            "未配置对话模型：CHAT_API_KEY 为空（端点 %s），AI 讲解不可用。"
            "请在 leetnote-ai/.env 里填对应平台的 key。",
            settings.effective_chat_base_url,
        )
        return None

    session_header = settings.chat_session_header.strip()
    if not session_header and settings.uses_opencode_go:
        # OpenCode Go 不带这个头会直接 503
        session_header = "x-opencode-session"

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
        session_header=session_header,
    )


__all__ = ["ChatClient", "ChatError", "ChatResult", "build_chat_client"]

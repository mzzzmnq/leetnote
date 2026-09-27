"""OpenAI 兼容的对话模型客户端。

针对 OpenCode Go 做了适配 —— 它对第三方客户端有两条硬性要求：
  1. 用【自己的】User-Agent 标识，不能是通用 SDK / HTTP 库的名字
  2. 每个会话带稳定的会话 ID 头，服务端据此做路由优化与 prompt 缓存
     （不带会直接 503：`Request is missing x-opencode-session`）

这两项都是可配置的，换成别的供应商留空即可自动跳过。

参考：https://opencode.ai/docs/go/#where-can-i-use-it
"""

from __future__ import annotations

import asyncio
import logging
from dataclasses import dataclass

import httpx

logger = logging.getLogger(__name__)

_MAX_RETRIES = 3


@dataclass(frozen=True, slots=True)
class ChatResult:
    text: str
    model: str
    prompt_tokens: int
    completion_tokens: int
    # 推理模型会把思考过程放在单独字段，这里一并保留便于排查
    reasoning: str = ""
    finish_reason: str = ""


class ChatError(RuntimeError):
    """调用对话模型失败。"""


class ChatClient:
    def __init__(
        self,
        *,
        base_url: str,
        api_key: str,
        model: str,
        user_agent: str = "",
        session_header: str = "",
        timeout: float = 120.0,
    ) -> None:
        self.model = model
        self._base_url = base_url.rstrip("/")
        self._api_key = api_key
        self._user_agent = user_agent
        self._session_header = session_header
        self._timeout = timeout

    def _headers(self, session_id: str) -> dict[str, str]:
        headers = {
            "Authorization": f"Bearer {self._api_key}",
            "Content-Type": "application/json",
        }
        if self._user_agent:
            headers["User-Agent"] = self._user_agent
        if self._session_header:
            # 同一会话重复请求必须带相同的值，否则永远命中不了 prompt 缓存
            headers[self._session_header] = session_id
        return headers

    async def complete(
        self,
        prompt: str,
        *,
        session_id: str,
        system: str = "",
        max_tokens: int = 4000,
        temperature: float = 0.3,
    ) -> ChatResult:
        """发一次对话请求。

        max_tokens 给得比较宽：推理模型（如 deepseek 系列）会先产出一大段
        reasoning，如果上限太小，正文会被挤成空字符串（completion_tokens
        正好等于上限就是被截断的信号）。
        """
        messages: list[dict[str, str]] = []
        if system:
            messages.append({"role": "system", "content": system})
        messages.append({"role": "user", "content": prompt})

        payload = {
            "model": self.model,
            "messages": messages,
            "max_tokens": max_tokens,
            "temperature": temperature,
        }

        last_error = "未知错误"

        for attempt in range(_MAX_RETRIES):
            if attempt:
                await asyncio.sleep(attempt * 2)

            try:
                async with httpx.AsyncClient(timeout=self._timeout) as client:
                    resp = await client.post(
                        f"{self._base_url}/chat/completions",
                        headers=self._headers(session_id),
                        json=payload,
                    )
            except httpx.HTTPError as exc:
                last_error = f"网络错误: {exc}"
                continue

            if resp.status_code >= 400:
                last_error = f"HTTP {resp.status_code}: {resp.text[:300]}"
                # 4xx 里只有 429 值得重试，其余重试也没用
                if resp.status_code != 429 and resp.status_code < 500:
                    break
                continue

            try:
                body = resp.json()
                choice = body["choices"][0]
                message = choice.get("message", {})
            except (KeyError, IndexError, ValueError) as exc:
                last_error = f"响应格式异常: {exc}"
                continue

            text = (message.get("content") or "").strip()
            reasoning = (message.get("reasoning_content") or message.get("reasoning") or "").strip()
            usage = body.get("usage") or {}
            finish = choice.get("finish_reason") or ""

            if not text:
                # 内容为空几乎总是被 max_tokens 截断了（推理吃光了配额）
                last_error = (
                    f"模型返回了空内容 (finish_reason={finish}, "
                    f"completion_tokens={usage.get('completion_tokens')})。"
                    f"若是推理模型，请调大 max_tokens。"
                )
                logger.warning("模型 %s %s", self.model, last_error)
                continue

            return ChatResult(
                text=text,
                model=body.get("model") or self.model,
                prompt_tokens=int(usage.get("prompt_tokens") or 0),
                completion_tokens=int(usage.get("completion_tokens") or 0),
                reasoning=reasoning,
                finish_reason=finish,
            )

        raise ChatError(f"调用对话模型失败（已重试 {_MAX_RETRIES} 次）: {last_error}")

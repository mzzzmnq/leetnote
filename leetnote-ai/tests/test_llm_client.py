"""对话模型客户端的单元测试。

重点验证 OpenCode Go 对第三方客户端的两条硬性要求：
  1. 必须带自己的 User-Agent（不能是通用 SDK 名）
  2. 必须带稳定的会话 ID 头 —— 不带会直接 503

用假的 httpx.AsyncClient 替换网络层，测试不依赖外网。
"""

from __future__ import annotations

import json

import httpx
import pytest

from app.llm.client import ChatClient, ChatError


class FakeResponse:
    def __init__(self, status_code: int, payload: dict | None = None, text: str = "") -> None:
        self.status_code = status_code
        self._payload = payload
        self.text = text or (json.dumps(payload) if payload else "")

    def json(self) -> dict:
        if self._payload is None:
            raise ValueError("响应不是合法 JSON")
        return self._payload


class FakeAsyncClient:
    """按顺序返回预设响应，并记录每次请求。"""

    def __init__(self, responses: list[FakeResponse], calls: list[dict]) -> None:
        self._responses = responses
        self._calls = calls

    async def __aenter__(self) -> FakeAsyncClient:
        return self

    async def __aexit__(self, *_: object) -> bool:
        return False

    async def post(self, url: str, headers: dict | None = None, json: dict | None = None) -> FakeResponse:
        self._calls.append({"url": url, "headers": headers or {}, "body": json or {}})
        return self._responses.pop(0)


async def _no_sleep(_seconds: float) -> None:
    """把重试退避跳过，测试不必真的等待。"""
    return None


def patch_httpx(monkeypatch: pytest.MonkeyPatch, responses: list[FakeResponse]) -> list[dict]:
    calls: list[dict] = []
    monkeypatch.setattr(
        httpx,
        "AsyncClient",
        lambda **_: FakeAsyncClient(responses, calls),
    )
    return calls


def ok_response(text: str = "好的", model: str = "space-bunny-free") -> FakeResponse:
    return FakeResponse(
        200,
        {
            "model": model,
            "choices": [{"message": {"content": text}, "finish_reason": "stop"}],
            "usage": {"prompt_tokens": 100, "completion_tokens": 50},
        },
    )


def client(**kwargs: object) -> ChatClient:
    defaults = {
        "base_url": "https://opencode.ai/zen/go/v1",
        "api_key": "oc_sk_test",
        "model": "space-bunny-free",
        "user_agent": "leetnote-ai/0.1.0",
        "session_header": "x-opencode-session",
    }
    defaults.update(kwargs)
    return ChatClient(**defaults)  # type: ignore[arg-type]


class TestHeaders:
    @pytest.mark.asyncio
    async def test_sends_user_agent_and_session(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        calls = patch_httpx(monkeypatch, [ok_response()])

        await client().complete("你好", session_id="leetnote-note-42")

        headers = calls[0]["headers"]
        # OpenCode Go 不带这两个头会直接 503
        assert headers["User-Agent"] == "leetnote-ai/0.1.0"
        assert headers["x-opencode-session"] == "leetnote-note-42"
        assert headers["Authorization"] == "Bearer oc_sk_test"

    @pytest.mark.asyncio
    async def test_omits_optional_headers_when_unset(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """换成非 OpenCode 的供应商时，这两个头应留空且不发送。"""
        calls = patch_httpx(monkeypatch, [ok_response()])

        await client(user_agent="", session_header="").complete("你好", session_id="x")

        headers = calls[0]["headers"]
        assert "User-Agent" not in headers
        assert "x-opencode-session" not in headers

    @pytest.mark.asyncio
    async def test_hits_chat_completions_path(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        calls = patch_httpx(monkeypatch, [ok_response()])

        await client().complete("你好", session_id="s")

        assert calls[0]["url"] == "https://opencode.ai/zen/go/v1/chat/completions"


class TestParsing:
    @pytest.mark.asyncio
    async def test_returns_text_and_usage(self, monkeypatch: pytest.MonkeyPatch) -> None:
        patch_httpx(monkeypatch, [ok_response("这是讲解")])

        result = await client().complete("讲一下", session_id="s")

        assert result.text == "这是讲解"
        assert result.model == "space-bunny-free"
        assert result.prompt_tokens == 100
        assert result.completion_tokens == 50

    @pytest.mark.asyncio
    async def test_system_prompt_prepended(self, monkeypatch: pytest.MonkeyPatch) -> None:
        calls = patch_httpx(monkeypatch, [ok_response()])

        await client().complete("讲一下", session_id="s", system="你是教练")

        messages = calls[0]["body"]["messages"]
        assert messages[0] == {"role": "system", "content": "你是教练"}
        assert messages[1]["role"] == "user"


class TestFailureHandling:
    @pytest.mark.asyncio
    async def test_empty_content_is_retried_then_errors(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """推理模型吃光 max_tokens 时会返回空内容，必须重试并最终明确报错。

        静默返回空字符串的话，用户点了按钮只会看到一个空白框，无从排查。
        """
        empty = FakeResponse(
            200,
            {
                "model": "deepseek-v4.1-flash",
                "choices": [{"message": {"content": ""}, "finish_reason": "length"}],
                "usage": {"prompt_tokens": 49, "completion_tokens": 200},
            },
        )
        monkeypatch.setattr("asyncio.sleep", _no_sleep)
        calls = patch_httpx(monkeypatch, [empty, empty, empty])

        with pytest.raises(ChatError) as exc:
            await client().complete("x", session_id="s")

        assert len(calls) == 3, "空内容应重试满 3 次"
        assert "max_tokens" in str(exc.value)

    @pytest.mark.asyncio
    async def test_retries_on_5xx(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setattr("asyncio.sleep", _no_sleep)
        patch_httpx(
            monkeypatch,
            [FakeResponse(503, text="overloaded"), ok_response("恢复了")],
        )

        result = await client().complete("x", session_id="s")

        assert result.text == "恢复了"

    @pytest.mark.asyncio
    async def test_does_not_retry_on_4xx(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """4xx 重试也没用，应立即失败，避免白白等待。"""
        calls = patch_httpx(monkeypatch, [FakeResponse(401, text="unauthorized")])

        with pytest.raises(ChatError):
            await client().complete("x", session_id="s")

        assert len(calls) == 1, "4xx 不应重试"

    @pytest.mark.asyncio
    async def test_network_error_is_wrapped(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setattr("asyncio.sleep", _no_sleep)

        class BoomClient(FakeAsyncClient):
            async def post(self, url: str, headers: dict | None = None, json: dict | None = None) -> FakeResponse:
                raise httpx.ConnectError("connection reset")

        monkeypatch.setattr(httpx, "AsyncClient", lambda **_: BoomClient([], []))

        with pytest.raises(ChatError) as exc:
            await client().complete("x", session_id="s")

        assert "网络错误" in str(exc.value)

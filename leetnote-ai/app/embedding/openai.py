"""OpenAI 兼容接口的 embedding 客户端。

兼容任何遵循 `POST /v1/embeddings` 的服务：
OpenAI、硅基流动、智谱、通义、DeepSeek 等。
换供应商只要改 base_url + model 两行配置。
"""

from __future__ import annotations

import asyncio
import logging

import httpx

logger = logging.getLogger(__name__)

# 各家的单次批量上限不同，取一个保守值
_BATCH_SIZE = 16
_MAX_RETRIES = 3


class OpenAIEmbedder:
    def __init__(
        self,
        *,
        base_url: str,
        api_key: str,
        model: str,
        dim: int,
        timeout: float = 30.0,
    ) -> None:
        self.name = model
        self.dim = dim
        self._base_url = base_url.rstrip("/")
        self._api_key = api_key
        self._model = model
        self._timeout = timeout

    async def embed(self, texts: list[str]) -> list[list[float]]:
        if not texts:
            return []

        out: list[list[float]] = []
        async with httpx.AsyncClient(timeout=self._timeout) as client:
            for i in range(0, len(texts), _BATCH_SIZE):
                batch = texts[i : i + _BATCH_SIZE]
                out.extend(await self._embed_batch(client, batch))

        # 维度不匹配要在写入数据库之前就报错，否则会得到一句难懂的
        # 「expected 1536 dimensions, not 1024」
        for vec in out:
            if len(vec) != self.dim:
                raise RuntimeError(
                    f"模型 {self._model} 返回的向量维度是 {len(vec)}，"
                    f"但配置里写的是 {self.dim}。"
                    f"请修改 EMBEDDING_DIM，并同步调整 note_embeddings 表的列定义。"
                )
        return out

    async def _embed_batch(
        self, client: httpx.AsyncClient, batch: list[str]
    ) -> list[list[float]]:
        last_error = "未知错误"

        for attempt in range(_MAX_RETRIES):
            if attempt:
                # 退避重试：外部 API 偶尔 429 / 502，重试通常就能成功
                await asyncio.sleep(attempt)

            try:
                resp = await client.post(
                    f"{self._base_url}/embeddings",
                    headers={"Authorization": f"Bearer {self._api_key}"},
                    json={"model": self._model, "input": batch},
                )
            except httpx.HTTPError as exc:
                last_error = f"网络错误: {exc}"
                continue

            if resp.status_code >= 400:
                last_error = f"HTTP {resp.status_code}: {resp.text[:200]}"
                continue

            try:
                payload = resp.json()
                items = payload["data"]
            except (KeyError, ValueError) as exc:
                last_error = f"响应格式异常: {exc}"
                continue

            # 按 index 排序，保证返回顺序与输入一致
            items = sorted(items, key=lambda item: item.get("index", 0))
            return [item["embedding"] for item in items]

        raise RuntimeError(f"调用 embedding 接口失败（已重试 {_MAX_RETRIES} 次）: {last_error}")

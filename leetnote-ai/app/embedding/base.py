"""Embedding 抽象。

用 Protocol 而不是抽象基类：调用方只依赖「有 name/dim 属性且能 embed」这个契约，
不需要继承任何东西，测试时替换成假实现也不用改类型。
"""

from __future__ import annotations

from typing import Protocol, runtime_checkable


@runtime_checkable
class Embedder(Protocol):
    """把文本转成定长向量。"""

    name: str
    """模型标识。会写进 note_embeddings.model —— 日后换了模型，
    能一眼看出某条向量是哪个模型产生的，避免新旧向量混在一起算相似度。"""

    dim: int
    """向量维度。必须与 note_embeddings.embedding 的列定义一致。"""

    async def embed(self, texts: list[str]) -> list[list[float]]:
        """批量向量化。返回顺序必须与输入一一对应。"""
        ...

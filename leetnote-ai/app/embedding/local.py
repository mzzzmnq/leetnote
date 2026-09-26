"""本地兜底向量器。

【定位】这是一个 **开发/测试用的占位实现**，不是语义模型。

它用「哈希技巧」（Hashing Trick）把文本的字/词 n-gram 映射到定长向量：
措辞相近的文本会得到相近的向量，足以把整条链路
（向量化 → 存库 → 相似度检索 → 接口 → 前端）跑通并测试。

但它 **不理解语义**：搜「动态规划」不会匹配到只写了「DP」的笔记。
真正上线需要配置 embedding 模型（见 openai.py）。
"""

from __future__ import annotations

import hashlib
import math
import re
from collections.abc import Iterator

# 英文/代码标识符 与 连续中文 分开处理
_TOKEN_RE = re.compile(r"[A-Za-z0-9_]+|[\u4e00-\u9fff]+")


def _hash_token(token: str) -> int:
    """稳定哈希。

    不能用内置 hash()：Python 默认开启哈希随机化（PYTHONHASHSEED），
    同一段文本在不同进程里会得到不同向量 —— 服务一重启，
    之前存的向量就全对不上了。
    """
    digest = hashlib.blake2b(token.encode("utf-8"), digest_size=8).digest()
    return int.from_bytes(digest, "big")


def tokenize(text: str) -> Iterator[str]:
    """切词。

    中文没有空格分隔，这里用「单字 + 相邻双字」：
    单字保证召回率，双字补充语序信息（「动态规划」和「规划动态」应该不同）。
    """
    for match in _TOKEN_RE.finditer(text.lower()):
        token = match.group()
        if token[0].isascii():
            yield token
        else:
            yield from token
            for i in range(len(token) - 1):
                yield token[i : i + 2]


class LocalHashingEmbedder:
    """基于哈希技巧的本地向量器。"""

    name = "local-hashing-v1"

    def __init__(self, dim: int = 1536) -> None:
        self.dim = dim

    async def embed(self, texts: list[str]) -> list[list[float]]:
        return [self._embed_one(t) for t in texts]

    def _embed_one(self, text: str) -> list[float]:
        vec = [0.0] * self.dim

        for token in tokenize(text):
            h = _hash_token(token)
            idx = h % self.dim
            # 有符号哈希：让不同 token 落到同一维时相互抵消一部分，
            # 而不是一味累加（Hashing Trick 的标准做法，能减小碰撞带来的偏差）
            sign = 1.0 if (h >> 63) & 1 else -1.0
            vec[idx] += sign

        # L2 归一化：让余弦相似度只反映方向，不受文本长短影响
        norm = math.sqrt(sum(v * v for v in vec))
        if norm == 0:
            return vec
        return [v / norm for v in vec]

"""本地哈希向量器的单元测试。

它虽然不是语义模型，但有几个必须成立的性质：
确定性、维度正确、归一化、字面相近的文本相似度更高。
这些性质一旦破坏，检索结果就会变得随机 —— 所以值得测。
"""

from __future__ import annotations

import math

import pytest

from app.embedding.local import LocalHashingEmbedder, tokenize


@pytest.fixture
def embedder() -> LocalHashingEmbedder:
    return LocalHashingEmbedder(dim=512)


def cosine(a: list[float], b: list[float]) -> float:
    return sum(x * y for x, y in zip(a, b, strict=True))


class TestTokenize:
    def test_splits_english_words(self) -> None:
        assert list(tokenize("two sum")) == ["two", "sum"]

    def test_chinese_yields_single_chars_and_bigrams(self) -> None:
        got = list(tokenize("哈希表"))
        # 单字：哈 希 表；双字：哈希 希表
        assert got == ["哈", "希", "表", "哈希", "希表"]

    def test_mixed_text(self) -> None:
        got = list(tokenize("用 hash 表"))
        # 英文整词保留，中文逐字 + 相邻双字
        assert "hash" in got
        assert "用" in got
        assert "表" in got
        assert "用表" not in got  # 被空格隔开，不该组成双字

    def test_lowercases(self) -> None:
        assert list(tokenize("TWO Sum")) == ["two", "sum"]

    def test_empty_text(self) -> None:
        assert list(tokenize("")) == []


class TestLocalHashingEmbedder:
    @pytest.mark.asyncio
    async def test_dimension(self, embedder: LocalHashingEmbedder) -> None:
        [vec] = await embedder.embed(["两数之和"])
        assert len(vec) == 512

    @pytest.mark.asyncio
    async def test_is_deterministic(self, embedder: LocalHashingEmbedder) -> None:
        """同样的文本必须得到同样的向量。

        这条最关键：如果用了 Python 内置 hash()，进程重启后哈希种子变化，
        之前入库的向量就全对不上了，检索结果会变得毫无意义。
        """
        [first] = await embedder.embed(["动态规划 背包问题"])
        [second] = await embedder.embed(["动态规划 背包问题"])
        assert first == second

    @pytest.mark.asyncio
    async def test_is_l2_normalized(self, embedder: LocalHashingEmbedder) -> None:
        [vec] = await embedder.embed(["滑动窗口与双指针"])
        norm = math.sqrt(sum(v * v for v in vec))
        assert norm == pytest.approx(1.0, abs=1e-9)

    @pytest.mark.asyncio
    async def test_empty_text_gives_zero_vector(self, embedder: LocalHashingEmbedder) -> None:
        [vec] = await embedder.embed([""])
        assert all(v == 0.0 for v in vec)

    @pytest.mark.asyncio
    async def test_batch_preserves_order(self, embedder: LocalHashingEmbedder) -> None:
        texts = ["两数之和", "二叉树", "滑动窗口"]
        vecs = await embedder.embed(texts)

        assert len(vecs) == 3
        # 逐条调用应与批量调用结果一致（顺序不能错位）
        for text, vec in zip(texts, vecs, strict=True):
            [single] = await embedder.embed([text])
            assert vec == single

    @pytest.mark.asyncio
    async def test_similar_text_scores_higher(self, embedder: LocalHashingEmbedder) -> None:
        """字面相近的文本，余弦相似度应明显高于不相关的文本。

        这是「哈希向量能当占位用」的前提 —— 否则整条链路测了也说明不了什么。
        """
        a_text = "两数之和：用哈希表记录已访问元素，把查找补数降到 O(1)"
        b_text = "两数之和的哈希表优化：遍历时边查边存，避免元素与自己配对"
        c_text = "二叉树层序遍历：用队列做 BFS，逐层记录当前层节点数"

        a, b, c = await embedder.embed([a_text, b_text, c_text])

        sim_ab = cosine(a, b)
        sim_ac = cosine(a, c)

        assert sim_ab > sim_ac, f"同主题应更相似: {sim_ab:.3f} vs {sim_ac:.3f}"
        assert sim_ab > 0.3, f"同主题相似度偏低: {sim_ab:.3f}"

    @pytest.mark.asyncio
    async def test_name_is_stable(self, embedder: LocalHashingEmbedder) -> None:
        """模型名会写进 note_embeddings.model，改了会导致新旧向量对不上。"""
        assert embedder.name == "local-hashing-v1"

    @pytest.mark.asyncio
    async def test_respects_configured_dim(self) -> None:
        big = LocalHashingEmbedder(dim=1536)
        [vec] = await big.embed(["测试"])
        assert len(vec) == 1536

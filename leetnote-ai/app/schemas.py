"""请求 / 响应模型。

字段命名用 snake_case，与 Go 服务和前端保持一致（避免一套系统两种命名风格）。
"""

from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field, field_validator


class EmbedRequest(BaseModel):
    note_id: int = Field(gt=0, description="要向量化的笔记 ID")


class EmbedResponse(BaseModel):
    note_id: int
    model: str
    dim: int
    ok: bool = True


class BatchEmbedRequest(BaseModel):
    """批量补向量：用于导入历史笔记或切换模型后重建。"""

    note_ids: list[int] = Field(default_factory=list, max_length=200)

    @field_validator("note_ids")
    @classmethod
    def _check_ids(cls, value: list[int]) -> list[int]:
        if any(v <= 0 for v in value):
            raise ValueError("note_ids 必须都是正整数")
        return value


class BatchEmbedResponse(BaseModel):
    requested: int
    succeeded: int
    failed: int
    model: str


class SimilarRequest(BaseModel):
    note_id: int = Field(gt=0)
    # user_id 由 Go 服务在鉴权后传入 —— 本服务不自己做用户认证，
    # 它只信任来自内部调用方的这个字段。
    user_id: int = Field(gt=0)
    limit: int = Field(default=5, ge=1, le=20)


class SimilarNoteOut(BaseModel):
    note_id: int
    title: str
    summary: str | None
    is_starred: bool
    similarity: float


class SimilarResponse(BaseModel):
    note_id: int
    model: str
    items: list[SimilarNoteOut]


class HealthResponse(BaseModel):
    status: str
    service: str
    embedding_model: str
    embedding_mode: Literal["remote", "local"]
    embedding_dim: int
    database: str

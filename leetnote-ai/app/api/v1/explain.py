"""LLM 解法讲解。

把笔记的正文与解法喂给对话模型，让它按固定结构点评并补充。
"""

from __future__ import annotations

import logging

from fastapi import APIRouter, Depends, HTTPException, Request, status

from app import repository
from app.api.deps import verify_internal_token
from app.llm.client import ChatClient, ChatError
from app.llm.prompts import EXPLAIN_SYSTEM, SolutionBrief, build_explain_prompt
from app.schemas import ExplainRequest, ExplainResponse

logger = logging.getLogger(__name__)

router = APIRouter(tags=["ai"], dependencies=[Depends(verify_internal_token)])


def get_chat_client(request: Request) -> ChatClient:
    """取对话模型客户端；未配置时明确报错而不是静默降级。"""
    client = getattr(request.app.state, "chat_client", None)
    if client is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="未配置对话模型（CHAT_API_KEY / CHAT_MODEL），LLM 讲解不可用",
        )
    return client


@router.post("/explain", response_model=ExplainResponse)
async def explain_note(
    payload: ExplainRequest,
    client: ChatClient = Depends(get_chat_client),
) -> ExplainResponse:
    material = await repository.fetch_note_for_explain(payload.note_id, payload.user_id)
    if material is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"笔记 {payload.note_id} 不存在或不属于该用户",
        )

    prompt = build_explain_prompt(
        title=material.title,
        problem_title=material.problem_title,
        difficulty=material.difficulty,
        content_md=material.content_md,
        solutions=[
            SolutionBrief(
                title=s.title,
                language=s.language,
                code=s.code,
                time_complexity=s.time_complexity,
                space_complexity=s.space_complexity,
            )
            for s in material.solutions
        ],
    )

    try:
        # 会话 ID 按笔记维度固定：同一篇笔记反复生成讲解时，
        # 前缀相同的 prompt 能命中服务端的 prompt 缓存，省 token 也更快。
        result = await client.complete(
            prompt,
            session_id=f"leetnote-note-{payload.note_id}",
            system=EXPLAIN_SYSTEM,
        )
    except ChatError as exc:
        raise HTTPException(
            status_code=status.HTTP_502_BAD_GATEWAY,
            detail=str(exc),
        ) from exc

    return ExplainResponse(
        note_id=payload.note_id,
        model=result.model,
        content=result.text,
        prompt_tokens=result.prompt_tokens,
        completion_tokens=result.completion_tokens,
    )

"""本机 OpenCode Go API key 自动发现。

背景：LeetNote 的「AI 讲解」由 OpenCode Go 订阅驱动。对**本机个人使用**而言，
在 .env 里手填 key 是多余的 —— opencode 自己就把 key 存在本地凭据库里，直接读即可。

查找顺序（先易后难）：
  1. ``~/.local/share/opencode/auth.json`` 里的 ``opencode-go.key``
  2. ``~/.local/share/opencode/opencode.db`` 的 ``credential`` 表
     （``integration_id LIKE 'opencode%' AND active = 1``，value 是
     JSON ``{"type":"key","key":"oc_sk_..."}``）

【安全】只读；找不到返回 ``None``，绝不抛异常影响启动。
【安全】日志只报告「发现/未发现」，**不打印 key 明文**。
"""

from __future__ import annotations

import json
import logging
import sqlite3
from pathlib import Path

logger = logging.getLogger(__name__)

_OPENCODE_DIR = Path.home() / ".local" / "share" / "opencode"
_DEFAULT_AUTH_JSON = _OPENCODE_DIR / "auth.json"
_DEFAULT_DB = _OPENCODE_DIR / "opencode.db"


def discover_local_api_key(
    auth_json: Path | None = None,
    db_path: Path | None = None,
) -> str | None:
    """从本机 opencode 凭据库发现 OpenCode Go 的 key；找不到返回 None。

    参数可覆盖默认路径，纯粹为了可测试（测试里传临时文件）。
    """
    key = _from_auth_json(auth_json or _DEFAULT_AUTH_JSON)
    if key:
        logger.info("已从本机 opencode auth.json 发现 OpenCode Go key（来源=自动发现）")
        return key

    key = _from_sqlite(db_path or _DEFAULT_DB)
    if key:
        logger.info("已从本机 opencode 凭据库发现 OpenCode Go key（来源=自动发现）")
        return key

    logger.info("本机未发现 OpenCode Go key（auth.json / opencode.db 里都没有）")
    return None


def _from_auth_json(path: Path) -> str | None:
    try:
        if not path.is_file():
            return None
        data = json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:  # noqa: BLE001 - 发现失败不该影响服务启动
        logger.warning("读取 opencode auth.json 失败: %s", exc)
        return None

    if not isinstance(data, dict):
        return None
    entry = data.get("opencode-go")
    if isinstance(entry, dict):
        return _clean(entry.get("key"))
    return None


def _from_sqlite(path: Path) -> str | None:
    try:
        if not path.is_file():
            return None
        # 只读打开：不干扰正在运行的 opencode
        conn = sqlite3.connect(f"file:{path.as_posix()}?mode=ro", uri=True, timeout=2.0)
    except Exception as exc:  # noqa: BLE001
        logger.warning("打开 opencode.db 失败: %s", exc)
        return None

    try:
        rows = list(
            conn.execute(
                "SELECT value FROM credential "
                "WHERE active = 1 AND integration_id LIKE 'opencode%' "
                "ORDER BY time_updated DESC"
            )
        )
    except Exception as exc:  # noqa: BLE001 - 上游表结构可能变化
        logger.warning("查询 opencode credential 失败: %s", exc)
        return None
    finally:
        conn.close()

    for (value,) in rows:
        key = _key_from_value(value)
        if key:
            return key
    return None


def _key_from_value(value: object) -> str | None:
    if not isinstance(value, str):
        return None
    try:
        obj = json.loads(value)
    except Exception:  # noqa: BLE001
        return None
    if isinstance(obj, dict):
        return _clean(obj.get("key"))
    return None


def _clean(value: object) -> str | None:
    if isinstance(value, str) and value.strip():
        return value.strip()
    return None

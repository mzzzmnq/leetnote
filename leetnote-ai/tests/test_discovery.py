"""OpenCode Go key 自动发现的单测。

全部用临时文件 / 临时 sqlite，**不会读取真实的 ~/.local/share/opencode**。
"""

from __future__ import annotations

import json
import sqlite3
from pathlib import Path

import pytest

from app.core.config import DEFAULT_CHAT_MODEL, Settings
from app.llm import build_chat_client, discovery


def _make_credential_db(path: Path, *, active: int, key: str) -> None:
    conn = sqlite3.connect(path)
    conn.execute(
        "CREATE TABLE credential "
        "(id TEXT, integration_id TEXT, value TEXT, active INTEGER, time_updated INTEGER)"
    )
    conn.execute(
        "INSERT INTO credential VALUES (?, ?, ?, ?, ?)",
        ("c1", "opencode-go", json.dumps({"type": "key", "key": key}), active, 1),
    )
    conn.commit()
    conn.close()


def test_key_from_value_parses_json() -> None:
    assert discovery._key_from_value('{"type":"key","key":"oc_sk_abc"}') == "oc_sk_abc"
    assert discovery._key_from_value("not json") is None
    assert discovery._key_from_value(123) is None
    assert discovery._key_from_value('{"type":"key"}') is None


def test_discover_from_auth_json(tmp_path: Path) -> None:
    auth = tmp_path / "auth.json"
    auth.write_text(json.dumps({"opencode-go": {"key": "oc_sk_fromauth"}}), encoding="utf-8")

    got = discovery.discover_local_api_key(auth_json=auth, db_path=tmp_path / "none.db")
    assert got == "oc_sk_fromauth"


def test_discover_from_sqlite(tmp_path: Path) -> None:
    db = tmp_path / "opencode.db"
    _make_credential_db(db, active=1, key="oc_sk_fromdb")

    got = discovery.discover_local_api_key(auth_json=tmp_path / "none.json", db_path=db)
    assert got == "oc_sk_fromdb"


def test_discover_ignores_inactive(tmp_path: Path) -> None:
    db = tmp_path / "opencode.db"
    _make_credential_db(db, active=0, key="oc_sk_off")

    assert discovery.discover_local_api_key(auth_json=tmp_path / "n.json", db_path=db) is None


def test_discover_returns_none_when_absent(tmp_path: Path) -> None:
    assert discovery.discover_local_api_key(
        auth_json=tmp_path / "a.json", db_path=tmp_path / "b.db"
    ) is None


OPENCODE_BASE = "https://opencode.ai/zen/go/v1"
ZHIPU_BASE = "https://open.bigmodel.cn/api/paas/v4"


def test_build_chat_client_discovers_key_for_opencode(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr("app.llm.discover_local_api_key", lambda *a, **k: "oc_sk_discovered")

    settings = Settings(chat_api_key="", chat_base_url=OPENCODE_BASE, chat_model="some-model")
    client = build_chat_client(settings)

    assert client is not None


def test_build_chat_client_does_not_discover_for_other_providers(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    called = {"n": 0}

    def fake(*args, **kwargs):  # noqa: ANN002, ANN003
        called["n"] += 1
        return "oc_sk_discovered"

    monkeypatch.setattr("app.llm.discover_local_api_key", fake)

    # 端点不是 OpenCode Go 时：没有 key 就是不可用，且【不能】去读 opencode 凭据
    settings = Settings(chat_api_key="", chat_base_url=ZHIPU_BASE, chat_model="glm-4.7-flash")
    assert build_chat_client(settings) is None
    assert called["n"] == 0, "非 OpenCode Go 端点不该自动发现 key"


def test_build_chat_client_default_model_is_zhipu(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr("app.llm.discover_local_api_key", lambda *a, **k: None)

    settings = Settings(chat_api_key="k", chat_base_url="", chat_model="")
    client = build_chat_client(settings)

    assert client is not None
    # 默认模型应当是智谱的免费模型
    assert client.model == DEFAULT_CHAT_MODEL == "glm-4.7-flash"


def test_build_chat_client_prefers_explicit_env_key(monkeypatch: pytest.MonkeyPatch) -> None:
    called = {"n": 0}

    def fake(*args, **kwargs):  # noqa: ANN002, ANN003
        called["n"] += 1
        return "oc_sk_discovered"

    monkeypatch.setattr("app.llm.discover_local_api_key", fake)

    settings = Settings(chat_api_key="zhipu_key", chat_model="some-model")
    client = build_chat_client(settings)

    assert client is not None
    assert called["n"] == 0, "显式提供了 key，就不该再去自动发现"


def test_build_chat_client_disabled_without_any_key(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr("app.llm.discover_local_api_key", lambda *a, **k: None)

    settings = Settings(chat_api_key="", chat_model="")
    assert build_chat_client(settings) is None


def test_opencode_gets_session_header_automatically(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr("app.llm.discover_local_api_key", lambda *a, **k: None)

    settings = Settings(
        chat_api_key="k", chat_base_url=OPENCODE_BASE, chat_model="m", chat_session_header=""
    )
    client = build_chat_client(settings)

    assert client is not None
    # OpenCode Go 不带这个头会 503，所以即使 .env 留空也要自动补上
    assert client._session_header == "x-opencode-session"


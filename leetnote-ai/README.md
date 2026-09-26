# leetnote-ai

LeetNote 的 AI 服务 —— 负责**向量化**与**相似题检索**。

> 完整设计见 [`../docs/DESIGN.md`](../docs/DESIGN.md)，环境说明见 [`../docs/ENVIRONMENT.md`](../docs/ENVIRONMENT.md)。

## 职责边界

| 服务 | 负责 | 不负责 |
|---|---|---|
| `leetnote-api`（Go） | 业务数据、**用户鉴权** | 不做向量计算 |
| `leetnote-ai`（Python，本服务） | 向量化、相似度检索 | **不做用户认证** |

因此本服务**只应由 Go 服务调用**，用 `X-Internal-Token` 拦住直接访问。
**不要把它的端口暴露到公网**。

## 快速开始

```powershell
# 1. 确保数据库和 Go 服务在跑
D:\dev\pg-start.ps1

# 2. 准备环境（注意：必须用官方 Python，不能用 Anaconda，见下方「已知问题」）
C:\Users\<你>\AppData\Local\Programs\Python\Python311\python.exe -m venv .venv
.\.venv\Scripts\pip.exe install fastapi "uvicorn[standard]" pydantic-settings `
    "psycopg[binary,pool]" httpx numpy -i https://mirrors.aliyun.com/pypi/simple/

# 3. 配置
Copy-Item .env.example .env

# 4. 启动（必须用 run.py，不能直接 uvicorn，见下方「已知问题」）
.\.venv\Scripts\python.exe run.py

# 5. 验证
curl http://127.0.0.1:8000/health
```

## 接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/health` | 健康检查（**不校验令牌**，供监控探活） |
| POST | `/api/v1/ai/embed` | 为单篇笔记生成/重建向量 |
| POST | `/api/v1/ai/embed/batch` | 批量补向量（导入历史笔记、换模型后重建） |
| GET | `/api/v1/ai/embedding/status?user_id=` | 向量覆盖率 |
| POST | `/api/v1/ai/similar` | 相似题检索 |

除 `/health` 外都需要请求头 `X-Internal-Token`。

## Embedding 的两条路径

| 模式 | 触发条件 | 说明 |
|---|---|---|
| **远程模型** | 配了 `LLM_API_KEY` + `EMBEDDING_MODEL` | 真正理解语义。任何 OpenAI 兼容服务都可以（硅基流动 / 智谱 / 通义 / OpenAI） |
| **本地兜底** | 上面留空 | 哈希技巧（字符 n-gram → 定长向量）。**只捕获字面相似度，不理解语义**，用于开发调试 |

切换只需改 `.env`，代码零改动。

> ⚠️ 换模型时注意维度：`note_embeddings.embedding` 当前是 `vector(1536)`。
> 换成维度不同的模型（如 `BAAI/bge-m3` 是 1024 维）需要先做一次迁移改列定义。

## 三个已知问题（都踩过）

### 1. 必须用 `run.py` 启动，不能直接 `uvicorn app.main:app`

Windows 上 asyncio 默认用 `ProactorEventLoop`，而 **psycopg3 的异步模式依赖
`add_reader`/`add_writer`，Proactor 没有这两个 API**。结果是连接池一个连接都建不起来，
启动时**静默卡死**，只在日志里反复刷：

```
Psycopg cannot use the 'ProactorEventLoop' to run in async mode.
```

试过但**不管用**的办法：
- 在 `app/main.py` 里 `set_event_loop_policy` → uvicorn 的顺序是「建循环 → 导入 app」，太晚
- 在 `run.py` 顶部 `set_event_loop_policy` → **uvicorn 0.53 已改用 loop factory**，
  内部硬编码返回 `ProactorEventLoop`，策略被完全绕过

`run.py` 的做法是**绕开 uvicorn 的 loop factory**，自己在 `asyncio.run()` 里跑
`Server.serve()`，事件循环就按当前策略创建了。

代价：`--reload` 用不了（那需要 uvicorn 的子进程 supervisor）。改代码手动重启即可。

### 2. 不能用 Anaconda 的 Python

Anaconda 附带的 **Python 3.11.7 配的是 OpenSSL 3.5.7**，与它编译时的版本不匹配，
导致任何 HTTPS 请求都失败：

```
SSLError: [ASN1: NOT_ENOUGH_DATA] not enough data (_ssl.c:4035)
```

表现为 `pip install` 报「找不到任何版本」，同时 `curl` 却一切正常 —— 很容易误判成网络问题。

**用官方 Python**（`AppData\Local\Programs\Python\Python311`，配 OpenSSL 3.0.11）即可。
诊断命令：

```powershell
python -c "import ssl; print(ssl.OPENSSL_VERSION)"
# Anaconda: OpenSSL 3.5.7   ← 有问题
# 官方:     OpenSSL 3.0.11  ← 正常
```

### 3. pip 换源

清华源会限流返回 403，用阿里云：

```powershell
pip install ... -i https://mirrors.aliyun.com/pypi/simple/
```

## 测试

```powershell
.\.venv\Scripts\python.exe -m pytest tests/ -v
```

## 目录结构

```
leetnote-ai/
├── run.py                    入口（设置事件循环策略后再启动 uvicorn）
├── pytest.ini
├── app/
│   ├── main.py               FastAPI 应用 + lifespan
│   ├── db.py                 psycopg 异步连接池
│   ├── repository.py         notes / note_embeddings 读写
│   ├── schemas.py            Pydantic 请求响应模型
│   ├── core/
│   │   ├── config.py         配置
│   │   └── logging.py
│   ├── embedding/
│   │   ├── base.py           Embedder 协议
│   │   ├── openai.py         OpenAI 兼容（真实路径）
│   │   └── local.py          本地哈希向量（兜底）
│   └── api/
│       ├── deps.py           内部令牌校验
│       └── v1/
│           ├── health.py
│           └── similar.py
└── tests/
    └── test_embedding.py
```

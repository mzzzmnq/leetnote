# LeetNote 学习导读（Go 后端 + Python AI）

> 面向「想重点吃透 Go 后端与 Python 部分」的学习者。
> 内容由**代码自身的注释与结构**整理而来 —— 真正的老师是代码，这份导读只是地图。

**怎么用**：按顺序读，每读一个文件，合上代码用嘴复述一遍；卡住就回到对应章节。
标记：🌟 = 必读（面试高频）｜⚙️ = 进阶/亮点。

**三遍法**：① 读懂在干嘛 → ② 用自己的话讲"为什么这么做" → ③ 动手改一点并跑测试。

---

# 第一部分 · Go 后端（`leetnote-api`）

## 阶段 0 · 入口与配置

### 🌟 `cmd/server/main.go`
- **作用**：进程入口。`main → run() error` 模式（`run` 返回 error 让 `defer` 能正常释放资源）。
- **关键**：`config.Load` → `logging` → `db.NewPool` → `cache.New` → `http.Server` → 优雅关闭（`signal.Notify` + `srv.Shutdown`）。
- **要掌握**：HTTP 各超时的取舍 —— `WriteTimeout` 特意放宽到 **120s**，因为 `POST /notes/:id/explain` 会同步等 LLM 十几秒。
- **自测**：为什么 `run` 不直接 `os.Exit`？`ReadHeaderTimeout` 防的是什么攻击？

### 🌟 `internal/config/config.go`
- **作用**：环境变量 → `Config` 结构体（`envconfig` 标签 + 默认值）。`Load()` 会尝试读 `.env`。
- **关键**：`IsProduction()`；`IsCookieSecure()`。
- **要掌握**：`IsCookieSecure` 的注释解释了一个经典坑 —— **`Secure` 属性取决于"是否走 HTTPS"，不是"是否生产环境"**，混用会导致 Docker 里 `http://localhost` 登录后刷新掉登录。
- **自测**：为什么配置要从环境变量来（12-Factor）？

### `internal/logging/logging.go`
- **作用**：`New(appEnv)` 返回 `*slog.Logger`（标准库结构化日志，零依赖）。生产环境输出 JSON。

---

## 阶段 1 · 一次请求的完整链路 ⭐ 最核心

**主线**：`router` → `middleware` → `handler` → `service` → `repository` → `pgx` → PostgreSQL

### 🌟 `internal/router/router.go`
- **作用**：**唯一装配点**。手工构造依赖注入（不用 wire/fx），注册中间件与路由。
- **关键**：`New(cfg, pool, cacheClient)`；`apiTokenRepo`；`rateCfg`；`r.NoRoute(...)` 统一错误格式。
- **要掌握**：**中间件顺序（洋葱模型）**：`RequestID → Logger → Recovery → CORS → [Auth → RateLimit] → handler`。为什么 RequestID 在最外层？（后面所有日志才带得上它）
- **自测**：为什么不让 handler 直接写 SQL？这层分层带来了什么、代价是什么？

### 🌟 `internal/middleware/requestid.go`
- 每个请求生成/复用 `X-Request-ID`，注入 context 并回写响应头。**全链路排查的刚需。**

### `internal/middleware/logger.go`
- 记录 `method/path/status/latency_ms/ip/request_id`（结构化）。

### `internal/middleware/recovery.go`
- `panic` → 500，并把堆栈写日志（不返回给客户端）。看懂"为什么不把 panic 详情返回"。

### `internal/middleware/cors.go`
- 允许来源白名单 + 预检（OPTIONS）。配合前端跨端口访问。

### 🌟 `internal/middleware/auth.go`
- **作用**：校验 `Authorization: Bearer`，把 `user_id` 注入 context。
- **关键**：`PATResolver` 接口；`Auth(tokens, pat)`；`GetUserID` / `MustUserID`。
- **要掌握**：**PAT 与 JWT 靠前缀分流**（`ln_` 走长期令牌，否则按 JWT 解析），且 JWT 显式要求 `TypeAccess`（refresh 不能直接访问业务接口）。
- **自测**：为什么中间件依赖一个**窄接口**而不是整个 repository？

### ⚙️ `internal/middleware/ratelimit.go`
- **作用**：基于 Redis 滑动窗口的限流；未登录按 IP、已登录按 user_id；超限 429 + `Retry-After` + `X-RateLimit-*`。
- **要掌握**：**fail-open** —— Redis 出错时放行并记日志。为什么？（限流是保护性能力，不该反过来打断正常请求）
- **自测**：`X-RateLimit-Remaining` 是怎么算出来的？

### `internal/handler/params.go`
- `paramInt64(c, "id")`：路径参数解析 + 统一错误响应。小函数，但能看出"错误处理收敛在一处"。

### 🌟 `internal/handler/note.go`
- **作用**：笔记相关 HTTP 层。`Create / Import / List / Get / Update / Delete / ToggleStar / ListSolutions / CreateSolution / UpdateSolution / DeleteSolution / Similar / Explain`。
- **要掌握**：handler **只做**「解析请求 → 调 service → 包装响应」，业务全在 service。
- **自测**：`Similar` 为什么用 POST 而不是 GET？两者语义差别？

### 🌟 `internal/service/note_service.go`
- **作用**：项目核心业务。
- **关键**：`Create` / `Update`（**事务**里同时写 notes/solutions/note_tags）、`enrichNotes`、`ImportNote` + `ensureProblem`、`scheduleEmbed`、`solutionFromInput`。
- **要掌握（面试金句级）**：
  1. `enrichNotes`：**批量查**（3 次固定查询）消除 **N+1**，与条数无关。
  2. `scheduleEmbed`：**异步**生成向量，且用**独立 context**（请求 context 会随响应取消）。
  3. `solutionFromInput`：解法有**两条写入路径**（单独接口 + 保存笔记整批替换），归一化只写一次、两条都走它 —— 避免"只改一条"的经典漏网。
  4. `ImportNote`：按 slug 关联/新建题目；同一题已有笔记则**更新**（幂等，不 409）。
- **自测**：事务边界应该画在哪？异步副作用失败怎么办（生产该怎么做）？

### ⚙️ `internal/repository/querier.go`
- **作用**：`Querier` 接口（`Query/QueryRow/Exec`）。**让仓储既能用连接池、也能用事务**——同套代码在事务内外复用。
- **要掌握**：这是 Go **隐式接口**的经典用法（`pgxpool.Pool` 和 `pgx.Tx` 都天然满足）。

### `internal/repository/note_repo.go`
- **关键**：`noteColumns`（用相关子查询带 `solution_count`，避免列表页再查一次）、`noteOrderBy`（**排序白名单**）、`GetByProblemID`、唯一键冲突 → 业务错误映射。
- **要掌握**：为什么 `ORDER BY` 绝不能拼接用户输入？（SQL 注入入口）

### 🌟 `internal/pkg/errs/errs.go`
- **作用**：可携带「业务错误码 + HTTP 状态码」的 `AppError`。
- **关键**：`Code/Message/HTTPStatus/Details/Err`；`Error/Unwrap/Is`；`Wrap/WithMessage/WithDetails`（**克隆**，不改全局变量）；预定义错误；`IsNotFound`。
- **要掌握**：
  1. `Is` 按 **Code** 比较（否则 `errors.Is(derived, ErrXxx)` 会因指针不同而失败）。
  2. `Wrap` 返回副本 —— 全局错误是共享模板，直接改会污染所有请求。
  3. 底层 `Err` 只写日志、绝不返回前端（防泄漏表结构）。
- **自测**：越权为什么返回 404 而不是 403？

### 🌟 `internal/pkg/response/response.go`
- **作用**：统一响应体 `{error:{code,message,details}}` 与 `OK/Created/NoContent/Page/Fail/Abort`。
- **要掌握**：`Fail` 用 `errors.As` 识别 `*AppError`；未识别的一律 500 且**只给笼统提示**；`NoContent` 要显式 `WriteHeaderNow`。

### `internal/validator/validator.go`
- `username` 正则校验；`BindJSON / BindQuery` 统一绑定与**字段级错误提示**。

### `internal/dto/*.go`
- 请求/响应模型 + `NewXxxResponse` 转换函数。**DTO 与领域模型分离**（`model` 是数据库形状，`dto` 是接口契约）。
- 重点看：`common.go`（`PageQuery.Normalize` 分页边界）、`note.go`（列表项**刻意不含** `content_md`，防响应体膨胀）。

---

## 阶段 2 · 数据与认证

### `internal/db/db.go` / `internal/db/tx.go`
- 连接池参数注释（`MaxConns=10` 因为 PG 默认 `max_connections=100`，要留给别人）。
- `WithTx(ctx, pool, fn)`：事务包装，出错自动回滚。

### 🌟 `internal/pkg/jwt/jwt.go`
- **关键**：`Claims`、`TokenPair`、`Manager`、`GeneratePair`、`sign`、`Parse(tokenStr, expectedType)`、`newJTI`。
- **要掌握**：access/refresh 用 `typ` 区分；`expectedType` 防止拿 refresh 当 access 用。
- **自测**：JWT 是无状态的，"登出"到底怎么实现？（提示：`jti` + 黑名单，见 DESIGN.md）

### `internal/pkg/hash/password.go`
- bcrypt；`MaxPasswordBytes = 72`（bcrypt 上限）；`DummyVerify()` **防时序侧信道**（用户不存在时也做一次假校验，避免通过耗时判断用户是否存在）。

### `internal/service/auth_service.go` + `internal/handler/auth.go`
- 注册/登录/刷新；**防账号枚举**（用户名错了和密码错了返回同样错误）；`setRefreshCookie/clearRefreshCookie`（httpOnly + SameSite）。

### `internal/repository/user_repo.go`
- 关键：`isUniqueViolation` / `isForeignKeyViolation` / `translateUserError` —— 把数据库错误翻译成业务错误。

### 🌟 `migrations/*.sql`（8 个）
- 看**每个迁移的头注释**，全是设计理由的浓缩：
  - `000002` 为什么**不**写 `CREATE EXTENSION`（pgvector 非 trusted，应用账号装不了，会卡成 dirty）
  - `000003` 大小写不敏感唯一（`Alice`/`alice` 不能同时注册）
  - `000005` 题目↔标签多对多（保住题单的"专题"信息）
  - `000007` 语言收敛到规范值 + `DO $$ ... RAISE EXCEPTION` 兜底
  - `000008` PAT 表（只存 sha256）
- **要掌握**：**迁移必须 up/down 成对**，只测 up 永远发现不了回滚失败。

### `internal/model/*.go`
- `language.go`（别名归一化，看 `NormalizeLanguage`）、`note.go`、`problem.go`、`review.go`、`tag.go`。

### 测试怎么读（很值钱）
- `internal/service/auth_service_test.go`：`fakeUserRepo` **内存假仓储**，不打数据库 —— 单元测试。
- `internal/router/auth_test.go`：`httptest` 跑**真实 HTTP + 真实数据库** —— 集成测试。
- `internal/pkg/jwt/jwt_test.go`：篡改签名、`none` 算法、过期等**安全用例**。

---

## 阶段 3 · 亮点与工程化 ⚙️

### 🌟 `internal/cache/redis.go` + `internal/cache/ratelimit.go`
- `redis.go`：客户端封装，**连不上就标记未启用**（不阻断启动）。
- `ratelimit.go`：**Lua 脚本**实现滑动窗口（ZSET + `ZREMRANGEBYSCORE/ZCARD/ZADD/PEXPIRE`）。
- **要掌握**：为什么"读-判断-写"必须原子？（拆成多条命令会有竞态，双双放行）
- 对照测试 `internal/cache/ratelimit_test.go`：用 **miniredis**，CI 不需要真 Redis。

### 🌟 `internal/repository/problem_repo.go`
- `problemOrderBy` 白名单；`GetBySlug`（导入关联）；`List` 的 `ILIKE` + `pg_trgm`。
- 配合 `internal/repository/like.go`：转义 `%` `_` 通配符 —— **搜 `100%` 不会命中全部**。

### `internal/pkg/apitoken/apitoken.go` + `repository/api_token_repo.go`
- `ln_` 前缀、sha256 存储、`Looks()` 分流；`ResolveUserID` 用一条 `UPDATE ... RETURNING` 同时校验并刷新 `last_used_at`。

### `internal/ai/client.go` + `internal/oauth/github.go`
- **外部依赖独立成包**：HTTP 调用、超时、错误处理隔离，service 只面对领域对象。
- 看 `ai/client.go` 如何带 `X-Internal-Token` 调 Python 服务。

### 🌟 `internal/service/review_service.go`（SM-2）
- `ApplySM2(state, rating)` 是**纯函数**（无 IO），所以能被 7 个单测穷尽边界。
- **要掌握**：把算法写成纯函数，是"可测试性"的正面示范。

### `internal/repository/stats_repo.go`
- `streakQuery`：连续打卡用 **gaps-and-islands**（日期减行号）在 **SQL 里算**，不在应用层循环。

### `cmd/*`
- `cmd/migrate`：**为什么把 golang-migrate 当库引入**（换机器免装 CLI）。
- `cmd/importer`：**为什么导入是 CLI 不是 HTTP 接口**（运维行为，需审阅）。
- `cmd/token`：PAT 生成/列出/吊销。

### 其余 handler/service/repository 速查
| 领域 | handler | service | repository |
|---|---|---|---|
| 用户 | `user.go` | `user_service.go` | `user_repo.go` |
| 题目 | `problem.go` | `problem_service.go` | `problem_repo.go` |
| 标签 | `tag.go` | `tag_service.go` | `tag_repo.go` |
| 复习 | `review.go` | `review_service.go` | `review_repo.go` |
| 统计 | `stats.go` | `stats_service.go` | `stats_repo.go` |
| 搜索 | `search.go` | `search_service.go` | `search_repo.go` |
| 解法 | （在 `note.go`） | （在 `note_service.go`） | `solution_repo.go` |
| OAuth | `oauth.go` | `oauth_service.go` | `oauth_repo.go` |

---

## Go 语言要点 → 代码位置

| 知识点 | 去哪看 |
|---|---|
| struct / 方法 / 指针接收者 | `model/*.go`、`pkg/errs/errs.go` |
| 隐式接口 | `repository/querier.go`、`service` 的 `CardScheduler`、`middleware.PATResolver` |
| error 链（`Is/As/Unwrap`） | `pkg/errs`、`pkg/response.Fail` |
| context 与超时 | `service.scheduleEmbed`、`db.NewPool`、`http.Server` 超时 |
| goroutine（异步副作用） | `note_service.scheduleEmbed`、`review_service.ScheduleEnsureCard` |
| 泛型 | `response.PageData[T]`、`response.Page[T]` |
| 事务 | `db.WithTx`、`note_service.Create` |
| 测试 | `httptest`（router）、假仓储（service）、`miniredis`（cache） |

## 阶段自测（Go）
1. 画出一个请求从 `router` 到数据库的完整链路，标出每层职责。
2. 说出 5 个"数据库层"的取舍（N+1、排序白名单、事务边界、唯一约束、pg_trgm）。
3. 解释 JWT 双 Token 的刷新流程，以及"登出"。
4. 解释限流为什么用 Lua、Redis 挂了为什么放行。
5. 独立写一个"新接口"：handler + service + repo 各一层，并补 router 集成测试。

---

# 第二部分 · Python AI（`leetnote-ai`）

> 定位：**只做向量化与相似度检索、LLM 调用，不做用户认证**（认证在 Go 侧）。
> 只监听内网，用 `X-Internal-Token` 拦住直接访问。

## 🌟 `run.py`
- **作用**：唯一正确的启动方式。**不能用 `uvicorn app.main:app`**。
- **要掌握（真实踩坑，面试好故事）**：Windows 的 asyncio 默认 `ProactorEventLoop`，而 **psycopg3 异步依赖 `add_reader/add_writer`**，Proactor 没有 → 连接池建不起来、启动静默卡死。跑到 app 里再 `set_event_loop_policy` **太晚**（uvicorn 先建循环再导入 app），所以要在 `run.py` 里自定义 loop factory。
- **自测**：为什么改在 `app/main.py` 里设置策略不管用？

## `app/main.py`
- FastAPI 应用装配：`lifespan` 里初始化连接池、embedder、`chat_client`；日志打印各自状态。
- **要掌握**：职责边界（Go 管业务与鉴权，本服务只管向量/LLM）。

## 🌟 `app/core/config.py`
- `Settings(BaseSettings)` 从 `.env`/环境变量读取；`use_remote_embedding`、`use_chat_model`、`effective_chat_base_url/model`（**空串回退默认**）、`is_production`。
- **要掌握**：为什么"空字符串会覆盖默认值"是个坑，以及怎么用 effective 属性解决。

## `app/core/logging.py`
- 标准库 logging（不引 loguru/structlog —— 日志量小，少一个依赖）。

## `app/api/deps.py`
- `get_embedder(request)`：从 `app.state` 取；`verify_internal_token`：用 `secrets.compare_digest` 常量时间比较。

## `app/schemas.py`
- Pydantic 请求/响应模型；字段用 **snake_case** 与 Go/前端保持一致。
- 看 `BatchEmbedRequest._check_ids`：**自定义校验**（去重/上限）。

## 阶段 2 · 两个业务接口

### `app/api/v1/health.py`
- 报告 embedding 模式、数据库状态、`chat_enabled/chat_model`（**以实际构建的 client 为准**，因为 key 可能是自动发现的）。

### 🌟 `app/api/v1/similar.py` + `app/repository.py`
- `repository.py`：只碰 `notes` 与 `note_embeddings` 两张表；`_to_vector_literal` 把向量转成 pgvector 字面量。
- **要掌握**：**只比较同模型产生的向量**（不同模型的向量不在同一空间）；按 `user_id` 隔离。

### 🌟 `app/api/v1/explain.py` + `app/llm/*`
- `explain.py`：取笔记 → 建 prompt → 调 `ChatClient.complete` → 返回四段讲解。
- `llm/client.py`：**OpenCode Go 的两条硬性要求**（自带 `User-Agent`、每会话带稳定 `x-opencode-session`，**不带会 503**）。
- `llm/prompts.py`：prompt **集中管理**（会反复改，属于配置而非常量）。
- `llm/discovery.py`：**本机 key 自动发现**（读 opencode 凭据库；只读、不打明文、失败不影响启动）。
- `llm/__init__.py`：`build_chat_client` 的 key 顺序（显式 env → 自动发现 → 不可用）。
- **自测**：为什么只比较同模型向量？`x-opencode-session` 是干嘛的？

## 阶段 3 · 数据层与测试

### `app/db.py`
- psycopg3 **异步连接池**。为什么不用 SQLAlchemy（只碰两张表，手写 SQL 更直观）。

### `app/embedding/*`
- `base.py`：用 `Protocol` 定义契约（不继承，测试可替换）。
- `local.py`：**本地哈希向量器**（Hashing Trick）—— 开发/测试**占位实现**，能让整条链路跑通，但**不理解语义**。
- `openai.py`：任何 OpenAI 兼容 `/v1/embeddings` 服务（换供应商只改两行配置）。

### 测试（`tests/`）
- `test_embedding.py`：验证向量器**确定性 / 维度 / 归一化 / 近文本更相似**（这些性质破了检索就变随机）。
- `test_llm_client.py`：用假 `httpx.AsyncClient` 验证**两条硬性要求**（UA、session 头），离线可跑。
- `test_discovery.py`：用**临时文件/临时 sqlite** 测自动发现，绝不读真实凭据库。

## Python 要点 → 代码位置

| 知识点 | 去哪看 |
|---|---|
| async/await | `repository.py`、`llm/client.py` |
| 类型注解 / Protocol | `embedding/base.py` |
| Pydantic / 校验 | `schemas.py`、`core/config.py` |
| FastAPI 依赖注入 | `api/deps.py`、`api/v1/*.py` |
| httpx 异步客户端 | `llm/client.py`、`embedding/openai.py` |
| pytest / pytest-asyncio | `tests/*`、`pytest.ini` |
| 读外部系统（SQLite） | `llm/discovery.py` |

---

# 第三部分 · 整体与面试

## 端到端数据流
```
浏览器(:5173)
   └─ HTTP ─> Go(:8080)  [鉴权 / 业务 / 统一错误]
                 ├─ pgx ─> PostgreSQL(:5432)   [数据 + pgvector + pg_trgm]
                 ├─ HTTP(+X-Internal-Token) ─> Python(:8000)  [向量 / LLM]
                 └─ Redis(:6379)               [限流]
```

## 面试必答清单（先自己答，再对照 `docs/DESIGN.md`）
1. `problems` 为什么独立成表，而非每用户一份？
2. 部分唯一索引 `WHERE problem_id IS NOT NULL` 解决什么？
3. 什么是 N+1？怎么发现、怎么解决？
4. 越权返回 404 还是 403？为什么？
5. JWT 无状态，"登出"怎么实现？
6. 限流为什么必须用 Lua/原子操作？
7. HNSW 与精确检索的区别？为什么向量检索用近似算法？
8. 服务间为什么用 HTTP 不用 gRPC？
9. 异步副作用（生成向量/复习卡）失败怎么办？生产怎么改进？
10. 迁移为什么要 up/down 成对？

## 6 周时间表（每天 1–1.5h）
| 周 | Go | Python |
|---|---|---|
| 1 | 阶段 0 + 1（请求链路） | `run.py` / `main.py` / `config` / `deps` / `schemas` |
| 2 | 阶段 2（数据与认证） | `similar` + `repository` |
| 3 | 阶段 2 收尾 + 读测试 | `explain` + `llm/*` |
| 4 | 阶段 3（缓存/限流/APIToken/AI 客户端） | `db` + `embedding` + 测试 |
| 5 | 复述全部模块 + 答面试清单 | 同左 |
| 6 | 动手：加一个小接口 + Docker 实测 | — |

## 检验标准（比"读完"更重要）
- [ ] 白板画出请求全链路
- [ ] 说出 ≥10 个设计取舍及其理由
- [ ] 独立写一个新接口（含测试），`go test ./... -cover` 绿
- [ ] 能讲 3 个真实踩坑（PowerShell 编码 / npm shim / IPv6 绑定 / ProactorEventLoop）
- [ ] `router` 覆盖率保持 99.8%

> 卡住时：先看文件**顶部注释**（"为什么"都在那），再看对应单测（单测就是"用法说明书"）。

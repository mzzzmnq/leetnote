# AGENTS.md · 项目上下文

> 给 AI 助手和新协作者。**开工前请先读完这一页**，再去看 `docs/` 里的细节文档。
>
> 最后更新：2026-10-08 · 对应提交 `bd8fdf5`

---

## 这是什么

**LeetNote** —— 一个 LeetCode 刷题笔记应用。个人项目，目标是作为求职作品集。

技术上是**前后端分离 + 双后端语言**：Go 主服务（业务与鉴权）+ Python AI 服务（向量检索与 LLM），
前端 Vue 3。数据库 PostgreSQL + pgvector。

| 指标 | 数值 |
|---|---|
| 代码规模 | Go 9,939 行 / 前端 5,708 行 / Python 1,364 行 / PowerShell 1,571 行（均按非空行） |
| 接口 | 41 个路由（3 探针 + 38 业务） |
| 数据库 | 12 张业务表 + `schema_migrations`，8 个版本化迁移 |
| 测试覆盖 | router **99.8%** · model 93.8% · pkg 87.5–93.3% · cache 85.4% · ai 75.4% · middleware 58.7% · service ~24% |
| 单测数量 | Go 若干 + Python 22 个 |

**当前数据状态**：2 个用户 / 10 篇笔记 / 18 个解法 / 174 道题 / 32 个标签 / 迁移版本 8。
（其中 `demo` 账号是演示数据，`mzzzmnq` 是本人的。）

---

## 快速上手

### 环境要求

| 工具 | 版本 | 备注 |
|---|---|---|
| Go | ≥ 1.24 | 实际用 1.27.1 |
| Python | **3.11（官方安装版）** | **绝对不要用 Anaconda 的**，见「已知的坑」 |
| Node.js | ≥ 20 | |
| PostgreSQL | ≥ 16 + pgvector | 或者用 Docker 只起数据库 |
| Redis | ≥ 5 | **可选**：只用于接口限流；没装则限流自动降级为放行，不影响启动 |

### 三条命令

```powershell
.\setup.ps1          # 首次：检查工具 → 生成 .env → 装依赖 → 建库 → 迁移 → 编译（幂等）
.\start-all.ps1      # 启动全部服务
.\status.ps1         # 查看状态
```

或者**双击 `LeetNote.bat`** 打开交互式菜单（启动/停止/重启/看日志/首次初始化）。

服务起来后打开 <http://localhost:5173>。

### 服务拓扑

| 服务 | 端口 | 启动方式 | 日志 |
|---|---|---|---|
| PostgreSQL | 5432 | `scripts/postgres.ps1`（pg_ctl 分离启动） | `<PGDATA>\server.log` |
| Redis | 6379 | `start-all.ps1`（redis-server，**可选**） | 控制台/文件 |
| Go 主服务 | 8080 | `bin\leetnote-api.exe` | `%TEMP%\leetnote-logs\api.out.log` |
| Python AI | 8000 | **`python run.py`** | `%TEMP%\leetnote-logs\ai.out.log` |
| 前端 | 5173 | `npm run dev` | `%TEMP%\leetnote-logs\web.out.log` |

---

## 改完之后怎么验证 ⭐

**这一节最重要。改动必须在提交前跑过。**

### 后端

```powershell
cd leetnote-api

# 格式与静态检查（必须零输出）
gofmt -l .            # 本机没把 Go 加进 PATH 的话，用 local.config.ps1 里那个路径
go vet ./...

# 全量测试 + 覆盖率
go test ./... -count=1 -cover
```

期望：全部 `ok`，且 **router 覆盖率保持 99.8%**（新增接口必须补集成测试）。

> 集成测试**跑在开发库上**（用 `.env` 里的 `DATABASE_URL`），不是独立测试库。
> 所以跑测试需要 PostgreSQL 在运行。清理逻辑已保证不留脏数据。

### 前端

```powershell
cd frontend
npm run typecheck     # vue-tsc，必须零错误
npm run build         # 生产构建
```

### AI 服务

```powershell
cd leetnote-ai
.\.venv\Scripts\python.exe -m pytest
```

### 界面改动

浏览器打开 <http://localhost:5173>。**注意**：如果桌面窗口不可见（最小化），
截图工具会失败；而且**不可见标签页里 `getComputedStyle` 返回的是陈旧值**，
验证主题/样式时要重新加载页面再读。

### 数据库改动

改迁移后必须验证 `up → down → up` 往返，以及新库能否从零建起来：

```powershell
cd leetnote-api
.\dev.ps1 migrate-version      # 看当前版本
.\dev.ps1 migrate-up           # 应用
.\dev.ps1 migrate-down -Steps 1
```

更彻底的验证方式（迁移集是否真能重建 schema）见 `docs/DESIGN.md` § 5.7。

---

## 架构速查

```
浏览器 ──► 前端 :5173 ──► Go :8080 ──► Python :8000 ──► PostgreSQL :5432
                            │                             ▲
                            └─────────────────────────────┘

前端【不】直接访问 AI 服务。Go 侧统一鉴权后转发，
Python 用 X-Internal-Token 拒绝外部请求。
```

### 目录速查

```
leetnote/
├── leetnote-api/          Go 主服务
│   ├── cmd/server/        入口
│   ├── cmd/migrate/       数据库迁移（golang-migrate 的封装）
│   ├── cmd/importer/      灵神题单导入
│   ├── migrations/        7 个版本化迁移（up/down 成对）
│   ├── scripts/           前置 SQL（装扩展）
│   └── internal/
│       ├── router/        ★ 40 个路由与依赖装配（唯一装配点）
│       ├── service/       业务逻辑（SM-2 算法在这里）
│       ├── repository/    数据访问（Querier 抽象，可复用事务）
│       ├── handler/       HTTP 层
│       ├── middleware/    RequestID / Logger / Recovery / CORS / Auth
│       ├── model/         领域模型 + 语言白名单
│       └── dto/           请求/响应结构
├── leetnote-ai/           Python AI 服务
│   ├── run.py             ★ 唯一正确的启动方式
│   └── app/{api,rag,llm,embedding}/
├── frontend/              Vue 3
│   └── src/{views,components,stores,styles,utils}/
├── deploy/                Docker 部署
├── scripts/               本机路径探测 + 服务清单 + PostgreSQL 启停
└── docs/                  详细文档（见文末地图）
```

### 几个关键设计决定

| 决定 | 理由 |
|---|---|
| Go + Python 双后端 | 向量库/LLM 生态在 Python，业务与并发在 Go。**是工程判断，不是炫技** |
| 服务间用 HTTP 不用 gRPC | 这个数据量下 HTTP 够用；引入 gRPC 要 protoc 工具链与两侧代码生成，收益不抵复杂度 |
| 中文检索用 `pg_trgm` 不用 `tsvector` | `tsvector` 中文要 zhparser 扩展，云数据库通常装不了 |
| 主题用 CSS 变量不用 Sass 变量 | Sass 编译期就被替换成字面量，运行时改不了；换肤必须是运行时的 |
| 语言枚举必须归一化 | 否则 `Python`/`python`/`PY` 混存，按语言分组时同一门语言会散成好几组 |

---

## 硬性约定 ⚠️

**违反了会出问题，不是风格问题。**

### 1. 服务必须用 `start-all.ps1` 启动，不要直接跑

```powershell
# ❌ 这样起，shell 一退服务就被带走
go run ./cmd/server
cd leetnote-ai; .\.venv\Scripts\python.exe -m uvicorn app.main:app

# ✅ 用封装好的
.\start-all.ps1
```

原因：直接在 shell 里跑，进程是 shell 的子进程。命令超时/关窗口/编辑器重启都会把它带走。
`start-all.ps1` 用 `Start-Process` 让进程真正脱离。PostgreSQL 同理（`scripts/postgres.ps1`）。

### 2. AI 服务必须用 `run.py` 启动

`uvicorn app.main:app` **不行** —— uvicorn 把 `ProactorEventLoop` 硬编码进了 loop factory，
与 psycopg3 的异步不兼容。`run.py` 里做了事件循环适配。

### 3. Python 虚拟环境必须用官方 Python

Anaconda 那套配的 OpenSSL 不兼容，会让 pip 的所有 HTTPS 请求失败，
表现为「pip 找不到任何包」，很容易误判成网络问题。

### 4. `requirements.txt` 不能写非 ASCII 字符

pip 用**系统区域编码**（中文 Windows 上是 GBK）读这个文件。
写中文注释会让它直接报：

```
UnicodeDecodeError: 'gbk' codec can't decode byte 0xa3
```

要加说明请写英文，或放到 `docs/`。

### 5. 迁移里不能写 `CREATE EXTENSION`

pgvector **不是 trusted 扩展**，只有超级用户能装。而迁移是用应用账号 `leetnote` 跑的，
写进去必然 `permission denied`，整个迁移卡住并变成 `dirty`。

职责划分：**装扩展 = 超级用户一次性**（`scripts/init-extensions.sql`），
**建表 = 应用账号可重复**（`migrations/`）。

### 6. 迁移必须 up/down 成对

只写 up 的话，回滚到底时会因为外键依赖失败。**只有跑完整往返才能发现**，只测 up 永远看不出来。

### 7. 新增语言要改三处

`internal/model/language.go` 的 `SupportedLanguages`（权威）、
`frontend/src/utils/languages.ts`（展示）、迁移里的 `CHECK` 约束（兜底）。
漏了任何一处都会不一致。

### 8. 排序 / 筛选参数必须白名单化

绝不能把用户输入拼进 `ORDER BY` —— 那是 SQL 注入的经典入口。
参考 `repository/problem_repo.go` 的 `problemOrderBy`。

### 9. 加接口要补集成测试

`internal/router` 的覆盖率是 **99.8%**，这是刻意维持的。新增路由必须补 httptest。

---

## 已知的坑

按踩到的频率排序。**详细记录在 `docs/ENVIRONMENT.md` 与各处的代码注释里。**

| 坑 | 现象 | 根因 |
|---|---|---|
| 容器里的 `localhost` | Docker 里连不上数据库 | 每个容器有独立网络命名空间，`localhost` 是容器自己。要用服务名 `db` |
| `Secure` cookie | 登录成功但刷新就掉登录 | `Secure` 属性应该看「是否走 HTTPS」，不是「是否生产环境」 |
| 便携版 PostgreSQL 的进程上下文 | 所有连接报 `0xC0000142` | `pg_ctl start` 没脱离 shell，postmaster 存活在受限上下文里。必须 `Start-Process` |
| 集成测试污染全局表 | 题目库出现一堆重复的「Two Sum」 | `problems`/`tags` 是全局表，不随用户级联删除，测试清理时漏了 |
| Vue 模板组件漏导入 | 构建不报错，只在浏览器控制台警告 | 未解析的 kebab-case 组件被当成自定义元素 |
| Vue Router 复用组件实例 | 只有路由参数变化时 `onMounted` 不再触发 | 从「相似题推荐」点进另一篇笔记时页面不刷新 |
| Naive 的 `n-form-item` label | 屏幕阅读器读不出输入框名称 | 渲染的 `<label>` 没有 `for`，和输入框没有程序化关联 |
| Naive 的 `aria-label` | 加在 `n-input` 上不生效 | 会被透传到外层 `div`，要用 `:input-props` |
| pip 读 requirements | `UnicodeDecodeError: 'gbk'` | 见上面硬性约定 #4 |
| PowerShell 变量名 | 类型转换报错 | **变量名不区分大小写**，`$Version` 和 `$version` 是同一个 |
| `.NET` 相对路径 | `Could not find file` | `[System.IO.File]::ReadAllText` 用的是进程 CWD，不是 PowerShell 的 `$PWD`。要用绝对路径 |
| 隐藏标签页的计算样式 | 主题验证结果不对 | Chromium 对不可见页面不做样式重算，返回陈旧值。重新加载页面再读 |

---

## 当前进度

### 已完成

M1 骨架 → M2 认证（含 GitHub OAuth）→ M3 核心 CRUD → M4/M5 前端与笔记功能 →
M6 搜索与统计 → 题单导入 → M7 相似题 + LLM 讲解 → M8 间隔重复（SM-2）→
题目库增强（难度分 / 题单顺序）→ 界面改版 + 夜间模式 → 多语言解法 → 可移植化 + Docker →
接口限流（Redis 滑动窗口）→ 长期令牌（PAT）+ 外部工具导入接口（openleet 浏览器助手对接）

### 待办

优先级从高到低：

1. **Redis 缓存**（M8 剩余）—— **接口限流已做**（滑动窗口 + Lua，见 `internal/middleware/ratelimit.go` 与 `internal/cache/`）；还剩「缓存热榜」（题目列表 / 统计缓存 + 防穿透）
2. **图片上传**（M8 剩余）—— 笔记里插图
3. **GraphRAG 知识点图谱**（M7 剩余）—— 原设计里有，未做
4. **部署上线** —— Docker 那套已经写好，但**从未实际构建过**（本机没装 Docker）
5. **CI/CD** —— GitHub Actions（lint → test → build）
6. **`golangci-lint`** —— `dev.ps1 lint` 已预留，工具未装

### 明确未验证的部分

诚实起见列出来，**不要当成已完成**：

- **Docker 相关文件全部没实测过**。三个 Dockerfile、两个 compose、nginx 配置
  都按最佳实践写了，但这台机器没装 Docker，无法构建验证。首次构建时请留意报错。
- `service` 层覆盖率只有 26.0%（大部分逻辑被 router 的集成测试覆盖了，但纯逻辑分支覆盖不足）
- `middleware` 覆盖率 36.0%
- 集成测试跑在**开发库**上，不是独立测试库（早期遗留问题）

---

## 文档地图

| 文档 | 什么时候看 |
|---|---|
| **`AGENTS.md`**（本文） | 开工前，每次 |
| `docs/DEPLOY.md` | 换机器 / 部署 / 排错 |
| `docs/DESIGN.md` | 要改数据库、改接口、改架构时。含完整 DDL、API 契约、迁移策略、员工号面试要点 |
| `docs/ENVIRONMENT.md` | 环境相关的问题（安装、路径、踩坑） |
| `docs/ROADMAP.md` | 想知道某个功能为什么这么做、以及 22 周学习路线 |
| `README.md` | 对外介绍（功能、技术栈、快速开始）。**改功能时记得同步** |
| `leetnote-api/README.md` | 只跑后端时 |
| `leetnote-ai/README.md` | 只跑 AI 服务时 |

### 改完功能记得同步这些

- 新功能 → `README.md` 的功能表 + `docs/ROADMAP.md` 的进度
- 新接口 → `docs/DESIGN.md` 的 API 章节
- 新踩的坑 → `docs/ENVIRONMENT.md` 或本文的「已知的坑」

---

## 给接手的助手：几条建议

1. **先跑 `.\status.ps1` 和测试**，确认基线是绿的再动手。机器重启后服务会全停，
   `.\start-all.ps1` 拉起来即可（PostgreSQL 首次启动可能要 20 秒以上，属正常）。
2. **改动前先读对应模块的注释**。这个项目的注释密度很高，很多「为什么不那样做」
   的推理都写在里面 —— 那些是踩过的坑，不是废话。
3. **不要顺手重构 imports 顺序或代码风格**，除非确实有问题。改动越小越好审。
4. **提交信息用中文，遵循约定式提交**（`feat(scope): ...` / `fix(scope): ...`），
   正文里解释**为什么**这么改，而不只是改了什么。
5. **不确定就问**，不要猜。这个项目有不少「看起来多余其实是必须的」设计。

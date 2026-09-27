# LeetNote · 算法练习笔记系统 — 设计方案

> 一个用于记录 LeetCode 刷题笔记的个人网站。前后端分离，**Go + Python 双服务**架构。
> 本文档是**开发前的完整设计**，包含技术选型理由、数据库设计、API 契约、目录结构、部署方案与面试要点。

---

## 目录

1. [项目定位](#1-项目定位)
2. [技术选型](#2-技术选型)
3. [系统架构](#3-系统架构)
4. [功能清单与里程碑](#4-功能清单与里程碑)
5. [数据库设计](#5-数据库设计)
6. [API 设计](#6-api-设计)
6.5 [前端主题系统与无障碍](#65-前端主题系统与无障碍)
7. [项目目录结构](#7-项目目录结构)
8. [认证方案](#8-认证方案)
9. [部署方案（零服务器起步）](#9-部署方案零服务器起步)
10. [学习路线](#10-学习路线)
11. [简历怎么写](#11-简历怎么写)
12. [面试自测题](#12-面试自测题)

---

## 1. 项目定位

**一句话**：一个支持 Markdown 笔记、多解法对比、标签检索、间隔重复复习，并由大模型辅助讲解与相似题推荐的 LeetCode 刷题知识库。

**为什么它能进简历**：

| 维度 | 普通"笔记网站" | 本项目 |
|---|---|---|
| 架构 | 单体 / 模板渲染 | 前后端分离 + **Go/Python 双服务**，gRPC 通信 |
| 认证 | 无 / Session | JWT 双 Token + 自动续期 |
| 数据库 | 只 CRUD | 范式设计 + 索引优化 + 全文检索 + **向量检索** |
| 差异化 | 无 | 间隔重复（SM-2）、**GraphRAG 知识点图谱**、统计图表 |
| 工程化 | 无 | Docker Compose、CI/CD、golangci-lint、pytest、迁移管理 |
| 语言 | 单一 | **双栈**：Go 写高并发服务，Python 写 AI 能力 |

**范围控制**：先做 MVP（能记录、能查、能看），再叠加亮点。**不要一开始就上 K8s / 消息队列**——面试官更看重你能否把一件事做扎实、讲清楚。

---

## 2. 技术选型

### 主服务 · Go（核心业务）

| 技术 | 版本 | 选它的理由 |
|---|---|---|
| **Go** | 1.27 | 语法极简（25 个关键字），并发模型天然适合 I/O 密集的 API 服务 |
| Gin | ^1.10 | 生态最成熟、资料最多的 Go Web 框架 |
| pgx | ^5 | PostgreSQL 高性能驱动，优于 `database/sql` |
| golang-migrate | ^4 | 数据库迁移版本管理（Go 生态的 Alembic） |
| golang-jwt | ^5 | JWT 签发 / 校验 |
| x/crypto/bcrypt | — | 密码哈希 |
| Viper / koanf | — | 配置管理 |
| zerolog / zap | — | 结构化日志 |
| testify | — | 单元测试断言 |
| golangci-lint | — | 静态检查（lint 全家桶） |

> **为什么主服务用 Go，而不是 Python 或 Java？**
> ① 学习成本只有 Java 的 1/3（约 3 个月 vs 10–12 个月）；
> ② goroutine 模型适合 I/O 密集的 API 服务；
> ③ 编译为单二进制，部署极简；
> ④ Go 岗位集中在大厂与云原生/基础设施，**质量高、避开 Java 的红海内卷**。

### AI 服务 · Python

| 技术 | 版本 | 选它的理由 |
|---|---|---|
| Python | 3.12 | AI 生态的唯一选择 |
| FastAPI | ^0.115 | 异步、Pydantic 校验、自动 OpenAPI 文档 |
| Pydantic | v2 | 请求/响应模型校验，天然类型安全 |
| httpx | ^0.27 | 异步 HTTP 客户端（调用 LLM API） |
| **pgvector** | — | 向量检索，**复用同一个 PostgreSQL**，不引入额外组件 |
| LangChain / LlamaIndex | 最新 | RAG 编排（**重原理，不要只会调 API**） |
| pytest | — | 单元测试 |

> **为什么不把 AI 服务也用 Go 写？** AI 生态（向量库客户端、LLM SDK、数据处理）几乎全在 Python。强行用 Go 会事倍功半。**双语言是工程判断，不是炫技。**

### 服务间通信

| 技术 | 状态 | 用途 |
|---|---|---|
| **HTTP/JSON** | ✅ 已实现 | Go ↔ Python 服务间调用，用 `X-Internal-Token` 做服务间鉴权 |
| gRPC + Protocol Buffers | 📋 计划中 | 强类型契约、两侧自动生成代码、二进制传输更省带宽 |

> **为什么先上 HTTP 而不是原计划的 gRPC**：引入 gRPC 要拉 protoc 工具链、
> 维护 `.proto`、两侧生成代码——对一个只有两个方法、QPS 很低的内部调用来说，
> 收益不足以抵消复杂度。**接口契约（见 6.3）已经写好了**，将来调用量上来了直接换即可，
> service 层不用改。

### 前端

| 技术 | 版本 | 选它的理由 |
|---|---|---|
| Vue | 3.5（Composition API） | 国内岗位最多，`<script setup>` 语法清爽 |
| TypeScript | ^5.6 | **硬门槛**，不写 TS 在简历筛选阶段就吃亏 |
| Vite | ^6 | 秒级冷启动 |
| Pinia | ^2 | Vue 官方状态管理 |
| Vue Router | ^4 | 路由 + 路由守卫（登录拦截） |
| Axios | ^1.7 | 拦截器统一处理 Token 与错误 |
| Naive UI | ^2 | 组件库，TypeScript 支持极好 |
| Markdown 渲染 | **markdown-it + highlight.js** | 轻量可控；highlight.js 按需注册语言，产物 174KB（默认全量是 1MB） |
| Markdown 编辑 | **自建分屏编辑器** | textarea + 实时预览。原计划的 Vditor 较重（~1MB）且样式侵入性强，自建更可控，后续可替换 |
| ECharts | ^5 | 刷题统计图表 |

> **前端定位：最小可用即可。** 简历重点是后端 + AI 的深度，前端只需做得干净、能跑通。

### 基础设施

| 用途 | 选型 | 备注 |
|---|---|---|
| 容器化 | Docker + Docker Compose | 一键起 4 个服务：api / ai / postgres / redis |
| 反向代理 | Nginx | 静态资源 + 按路径分流到 Go / Python |
| CI/CD | GitHub Actions | lint → test → build → deploy |
| 代码规范 | **golangci-lint**（Go）+ Ruff（Python）+ ESLint/Prettier（前端） | 提交前自动检查 |

---

## 3. 系统架构

```mermaid
flowchart LR
    subgraph Browser["浏览器"]
        SPA["Vue 3 SPA<br/>(TypeScript + Pinia)"]
    end

    subgraph Edge["边缘 / 静态托管"]
        CDN["Vercel / Cloudflare Pages<br/>静态资源 CDN"]
    end

    subgraph Server["应用服务"]
        NG["Nginx<br/>反向代理 + 路径分流"]
        API["leetnote-api<br/>Go + Gin<br/>业务 API"]
        AI["leetnote-ai<br/>Python + FastAPI<br/>RAG / LLM"]
    end

    subgraph Data["数据层"]
        PG[("PostgreSQL 18<br/>+ pgvector")]
        RD[("Redis")]
        R2[("对象存储<br/>图片")]
    end

    LLM["大模型服务<br/>OpenAI / 通义 / DeepSeek"]

    SPA -->|HTTPS| CDN
    SPA -->|"REST /api/v1/*"| NG
    NG -->|"业务接口"| API
    NG -->|"/api/v1/ai/*"| AI
    API -->|"HTTP + 内部令牌"| AI
    API --> PG
    API --> RD
    API -->|预签名上传| R2
    AI --> PG
    AI -->|LLM API| LLM
```

**职责边界（面试会问"为什么这样拆"）**：

| 服务 | 负责 | 不负责 |
|---|---|---|
| `leetnote-api` (Go) | 用户、认证、题目、笔记、解法、标签、统计的 CRUD；高并发读写 | 不做 LLM 调用、不做向量计算 |
| `leetnote-ai` (Python) | 相似题检索（pgvector）、解法讲解生成、复习卡生成、GraphRAG | 不做用户体系、不直接对前端暴露全部接口 |

**一次典型请求**（新建笔记）：

1. 前端 `NoteEditView` 校验表单 → `POST /api/v1/notes`
2. Axios 拦截器注入 `Authorization: Bearer <access_token>`
3. Nginx 按路径分流 → `leetnote-api` (Go)
4. Go：JWT 中间件解 Token → 参数绑定与校验 → Service 层业务逻辑 → pgx 写 PostgreSQL
5. 返回 `NoteRead`，HTTP 201
6. 前端更新 Pinia store → 跳转详情页

**一次 AI 请求**（生成解法讲解）：

1. 前端 → `POST /api/v1/ai/explain`，body 含 `note_id`
2. Nginx 分流 → `leetnote-ai` (Python)
3. Python：pgvector 检索相似题与相关笔记 → 组装 prompt → 调用 LLM API
4. 返回结构化结果（思路 / 复杂度分析 / 易错点）

---

## 4. 功能清单与里程碑

### MVP（必须完成）

- [ ] 用户注册 / 登录 / 登出
- [ ] 题目的增删改查（题号、标题、难度、链接）
- [ ] 笔记的增删改查（Markdown 内容）
- [ ] 一个笔记下挂多个解法（语言、代码、时间复杂度、空间复杂度）
- [ ] 标签系统（算法标签：动态规划、双指针……）
- [ ] 列表分页、按难度/标签/关键字筛选
- [ ] Markdown 渲染 + 代码高亮

### 进阶（简历亮点）

- [ ] 间隔重复复习（SM-2 算法）与"今日待复习"
- [ ] 全文检索（PostgreSQL `pg_trgm` + GIN 索引）
- [ ] 统计看板（ECharts：累计刷题量、难度分布、近 30 天趋势、连续打卡）
- [ ] **AI 解法讲解**（LLM）
- [ ] **相似题推荐**（pgvector 向量检索）
- [ ] **知识点图谱 + GraphRAG**（借力图算法科研）
- [ ] 一键导入 LeetCode 题目元数据（用开源数据集，不爬站）
- [ ] 图片上传（粘贴截图）
- [x] **GitHub OAuth 登录**（支持账号关联与解绑）
- [ ] Redis 缓存热榜 / 接口限流

### 里程碑

| 阶段 | 交付物 | 验收标准 |
|---|---|---|
| M0 设计 | 本文档 | 表结构、API 契约定稿 |
| M1 Go 骨架 | `leetnote-api` 可跑通 | `/health` 返回 200，配置/日志/错误处理成型 |
| M2 认证 | 注册登录 | 拿到 Token 能访问 `/users/me` |
| M3 核心 API | 笔记/解法/标签 CRUD | `go test` 覆盖率 > 60% |
| M4 前端骨架 | 登录 + 列表 + 详情 | 能完整记录一篇笔记 |
| M5 检索与统计 | 搜索 + 图表 | 关键字搜索 < 200ms（有前后对比数据） |
| M6 上线 | 公网可访问 | HTTPS + CI 自动部署 + 压测数据 |
| M7 AI 服务 | 双服务打通 | 相似题检索有评估数据，LLM 讲解可用 |
| M8 GraphRAG | 知识点图谱 | 能按前置知识推荐复习顺序 |

---

## 5. 数据库设计

> 这一章是**补数据库短板**的重点。每张表后面都有"为什么这样设计"的说明，建议边看边在 DBeaver 里把表建出来。

### 5.1 实体关系图

```mermaid
erDiagram
    users     ||--o{ notes         : "撰写"
    problems  ||--o{ notes         : "对应题目"
    notes     ||--o{ solutions     : "包含多个解法"
    notes     ||--o{ note_tags     : "打标签"
    tags      ||--o{ note_tags     : "被引用"
    notes     ||--o| review_cards  : "生成复习卡"
    review_cards ||--o{ review_logs : "记录复习历史"

    users {
        bigint id PK
        varchar username UK
        varchar email UK
        varchar password_hash
    }
    problems {
        bigint id PK
        int leetcode_id
        varchar title
        varchar title_slug UK
        varchar difficulty
    }
    notes {
        bigint id PK
        bigint user_id FK
        bigint problem_id FK
        varchar title
        text content_md
        varchar status
        bool is_starred
    }
    solutions {
        bigint id PK
        bigint note_id FK
        varchar title
        varchar language
        text code
    }
    tags {
        bigint id PK
        varchar name UK
        varchar slug UK
        varchar kind
    }
    note_tags {
        bigint note_id PK,FK
        bigint tag_id PK,FK
    }
    review_cards {
        bigint id PK
        bigint note_id UK,FK
        real ease_factor
        int interval_days
        int repetitions
        timestamptz due_at
    }
    review_logs {
        bigint id PK
        bigint card_id FK
        smallint rating
        timestamptz reviewed_at
    }
```

### 5.2 建表 DDL（PostgreSQL）

```sql
-- ============ 用户 ============
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    username      VARCHAR(50)  NOT NULL UNIQUE,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    avatar_url    TEXT,
    bio           VARCHAR(200),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ============ 题目（全局共享的元数据） ============
CREATE TABLE problems (
    id          BIGSERIAL PRIMARY KEY,
    leetcode_id INTEGER,
    title       VARCHAR(200) NOT NULL,
    title_slug  VARCHAR(200) NOT NULL UNIQUE,   -- 如 two-sum，用于幂等导入
    difficulty  VARCHAR(10)  NOT NULL
                CHECK (difficulty IN ('Easy', 'Medium', 'Hard')),
    url         TEXT,

    -- 社区统计的难度分（000006 迁移加入）。
    -- 官方三档难度粒度太粗——同为 Medium，1400 分和 2400 分完全不是一个量级。
    -- 来自 zerotrac/leetcode_problem_rating，NULL 表示该题早于竞赛时代、无数据。
    rating      REAL,

    -- 在灵神题单里的原始顺序（000006 迁移加入）。
    -- 题单是从易到难编排的，这个顺序本身携带信息；按题号排会打乱难度曲线。
    -- 0 表示不属于任何题单（手工录入的题）。
    sort_order  INTEGER NOT NULL DEFAULT 0,

    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_problems_difficulty ON problems (difficulty);
CREATE INDEX idx_problems_rating     ON problems (rating);
CREATE INDEX idx_problems_sort_order ON problems (sort_order);

-- ============ 标签 ============
CREATE TABLE tags (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(50) NOT NULL UNIQUE,     -- 展示名，如「动态规划」
    slug       VARCHAR(50) NOT NULL UNIQUE,     -- 机器名，如 dynamic-programming
    kind       VARCHAR(20) NOT NULL DEFAULT 'algorithm'
               CHECK (kind IN ('algorithm', 'data_structure', 'topic')),
    -- 题单里的专题顺序（000006 迁移加入），用于还原「相向双指针 → 滑动窗口 → 二分」
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============ 笔记 ============
CREATE TABLE notes (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    problem_id BIGINT          REFERENCES problems(id) ON DELETE SET NULL,
    title      VARCHAR(200) NOT NULL,
    content_md TEXT NOT NULL DEFAULT '',
    summary    VARCHAR(500),
    status     VARCHAR(10) NOT NULL DEFAULT 'draft'
               CHECK (status IN ('draft', 'published')),
    is_starred BOOLEAN NOT NULL DEFAULT FALSE,
    view_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_notes_user_created ON notes (user_id, created_at DESC);
CREATE INDEX idx_notes_problem      ON notes (problem_id);
-- 同一用户对同一题目只允许一篇笔记（但允许不关联题目的自由笔记）
CREATE UNIQUE INDEX uq_notes_user_problem
    ON notes (user_id, problem_id) WHERE problem_id IS NOT NULL;

-- ============ 解法 ============
CREATE TABLE solutions (
    id               BIGSERIAL PRIMARY KEY,
    note_id          BIGINT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    title            VARCHAR(100) NOT NULL,     -- 如「哈希表 · 一次遍历」
    language         VARCHAR(20)  NOT NULL,     -- python / go / java / cpp ...
    code             TEXT NOT NULL,
    time_complexity  VARCHAR(50),               -- O(n)
    space_complexity VARCHAR(50),               -- O(1)
    sort_order       INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_solutions_note ON solutions (note_id, sort_order);

-- ============ 笔记 ↔ 标签（多对多） ============
CREATE TABLE note_tags (
    note_id BIGINT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    tag_id  BIGINT NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    PRIMARY KEY (note_id, tag_id)
);
CREATE INDEX idx_note_tags_tag ON note_tags (tag_id);

-- ============ 复习卡（M8 会用到，SM-2 算法） ============
CREATE TABLE review_cards (
    id               BIGSERIAL PRIMARY KEY,
    note_id          BIGINT NOT NULL UNIQUE REFERENCES notes(id) ON DELETE CASCADE,
    ease_factor      REAL    NOT NULL DEFAULT 2.5,   -- 难度系数
    interval_days    INTEGER NOT NULL DEFAULT 0,     -- 当前间隔（天）
    repetitions      INTEGER NOT NULL DEFAULT 0,     -- 连续答对次数
    due_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_reviewed_at TIMESTAMPTZ
);
CREATE INDEX idx_review_cards_due ON review_cards (due_at);

CREATE TABLE review_logs (
    id            BIGSERIAL PRIMARY KEY,
    card_id       BIGINT NOT NULL REFERENCES review_cards(id) ON DELETE CASCADE,
    rating        SMALLINT NOT NULL CHECK (rating BETWEEN 0 AND 5),
    prev_interval INTEGER,
    next_interval INTEGER,
    reviewed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_review_logs_card ON review_logs (card_id, reviewed_at DESC);

-- ============ 第三方账号关联（GitHub OAuth） ============
-- password_hash 改为可空：OAuth 注册的用户没有密码
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

CREATE TABLE oauth_accounts (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider       VARCHAR(20)  NOT NULL,   -- github
    provider_uid   VARCHAR(100) NOT NULL,   -- provider 侧用户唯一 ID
    provider_login VARCHAR(100),            -- provider 侧登录名（展示用）
    avatar_url     TEXT,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_oauth_provider_uid UNIQUE (provider, provider_uid)
);
CREATE INDEX idx_oauth_accounts_user ON oauth_accounts (user_id);

-- ============ 题目 ↔ 专题/知识点（多对多） ============
-- 题单是按专题组织的（相向双指针、滑动窗口、二分…），
-- 一道题可能同时属于多个专题，所以用关联表而不是在 problems 上加单列。
CREATE TABLE problem_tags (
    problem_id BIGINT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    tag_id     BIGINT NOT NULL REFERENCES tags(id)     ON DELETE CASCADE,
    PRIMARY KEY (problem_id, tag_id)
);
CREATE INDEX idx_problem_tags_tag ON problem_tags (tag_id);
```

### 5.2.1 题单导入（灵神题单）

```bash
cd leetnote-api

# 干跑：只解析不写库，先看统计
go run ./cmd/importer -file D:\dev\_downloads\lingshen-tidan.md -dry-run

# 正式导入（会调 LeetCode 接口补难度 + 拉社区难度分）
go run ./cmd/importer -file D:\dev\_downloads\lingshen-tidan.md

# 试跑前 20 道
go run ./cmd/importer -file xxx.md -max 20

# 网络不通时用本地难度分文件（GitHub 在国内不稳，见下）
go run ./cmd/importer -file xxx.md -ratings-file D:\dev\_downloads\ratings.txt
```

**为什么需要单独导入**：LeetCode 官方接口只给「英文标题 + slug + 难度」，
灵神的题单给的是「中文标题 + 专题归属 + **专题内顺序**」。两者**按 slug 关联**，
合并后才能得到「中文标题 + 难度 + 难度分 + 专题 + 顺序」的完整数据。

**工具是幂等的**：重复执行只更新已有记录，不产生重复数据。

导入时会补齐题单里没有的四样东西：

| 字段 | 来源 | 落地方式 |
|---|---|---|
| `difficulty` | LeetCode GraphQL（4064 道题一次拉完） | 按 slug 匹配，匹配不到用 `-fallback-difficulty` |
| `rating` | `zerotrac/leetcode_problem_rating` | 按 slug 匹配 |
| `url` | 由 slug 拼出 | `https://leetcode.cn/problems/<slug>/` |
| `sort_order` | 题单本身 | 解析时的行序号；标签侧同理取专题序号 |

两个容易踩的坑，代码里都做了处理：

- **难度分写入用 `COALESCE(EXCLUDED.rating, problems.rating)`**：难度分是外部数据，
  拉取失败时不能让 NULL 把库里已有的值覆盖掉。实测这个设计救过一次——
  重跑导入时 CDN 正好超时，73 条难度分一条没丢。
- **难度同样不能降级**：`-skip-fetch`（或 LeetCode 接口挂了）时拉不到难度，会用
  `-fallback-difficulty` 填占位值。如果直接写库，库里原本正确的难度会被这个占位值
  覆盖 —— 实测一次 `-skip-fetch` 就把 172 道题的难度全冲成了 Medium（接雨水、
  最小覆盖子串本来是 Hard）。所以写库前先查一次已有难度，占位值遇到已有题目就沿用旧值。
- **复用已有标签时也要 `UpdateMeta`**：题单里的「滑动窗口」「二分查找」很可能和
  用户手工建的同名标签撞名。按名字复用时如果直接 `continue`，那个标签的
  `sort_order` 会一直是 0，专题排序整列失效。

> 这三条是同一个原则：**导入是「用新数据补充」，不是「用本次结果覆盖」**。
> 任何一次外部依赖失败，都不该让库里已有的正确数据变差。

> **难度分的覆盖边界**：数据是从**竞赛**表现反推的，而 LeetCode 竞赛从 2018 年
> （约 700 多题）才开始，所以早期经典题（1. 两数之和、15. 三数之和、42. 接雨水）
> 没有分数。实测题单覆盖 73/172 ≈ 42%，缺的全部是竞赛时代之前的老题。
> 灵神推荐的 `huxulm/lc-rating` 插件数据也完全来自这里（已逐条比对确认无额外覆盖），
> 所以这是免费数据源的上限，不是实现问题。

> 数据来源：[EndlessCheng/codeforces-go · leetcode/README.md](https://github.com/EndlessCheng/codeforces-go/blob/master/leetcode/README.md)
> 难度分来源：[zerotrac/leetcode_problem_rating](https://github.com/zerotrac/leetcode_problem_rating)

### 5.3 中文全文检索

PostgreSQL 内置的 `to_tsvector` 对中文分词支持很差（需要 `zhparser` 扩展，云数据库通常装不了）。
**实用方案：`pg_trgm` 三元组索引 + `ILIKE`**，对中文子串匹配效果够用：

```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_notes_title_trgm   ON notes USING GIN (title gin_trgm_ops);
CREATE INDEX idx_notes_content_trgm ON notes USING GIN (content_md gin_trgm_ops);

-- 查询示例
SELECT id, title, similarity(title, '动态规划') AS score
FROM notes
WHERE title ILIKE '%动态规划%'
   OR content_md ILIKE '%动态规划%'
ORDER BY score DESC
LIMIT 20;
```

> 进阶（可选）：数据量上万后接入 **Meilisearch / Typesense**，体会"搜索引擎"和"数据库"的边界。

### 5.4 向量检索（M7 AI 服务用）

```sql
CREATE EXTENSION IF NOT EXISTS vector;

-- 笔记的向量表示（1536 维对应常见的 text-embedding 模型，按实际模型调整）
CREATE TABLE note_embeddings (
    note_id    BIGINT PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE,
    embedding  vector(1536) NOT NULL,
    model      VARCHAR(64)  NOT NULL,   -- 记录用了哪个 embedding 模型
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- HNSW 索引：近似最近邻，查询快
CREATE INDEX idx_note_embeddings_hnsw
    ON note_embeddings USING hnsw (embedding vector_cosine_ops);

-- 相似题检索
SELECT n.id, n.title, 1 - (e.embedding <=> $1) AS similarity
FROM note_embeddings e
JOIN notes n ON n.id = e.note_id
ORDER BY e.embedding <=> $1
LIMIT 10;
```

> `<=>` 是余弦距离运算符，`1 - 距离` 即相似度。**这是把"AI 能力"落到"数据库技能"上的绝佳面试素材。**

### 5.5 设计决策说明（面试会问）

| 决策 | 理由 |
|---|---|
| `problems` 全局共享，而不是每个用户存一份 | 题目元数据与用户无关，重复存储违反 3NF；导入 3000 道题也只存一份 |
| `notes.problem_id` 允许 NULL | 支持"算法专题总结"这类不绑定具体题目的笔记 |
| 部分唯一索引 `WHERE problem_id IS NOT NULL` | 既保证"一题一笔记"，又不阻止多条 NULL |
| 多解法拆 `solutions` 表而非 JSON 字段 | 需要按语言/复杂度筛选排序，独立成表可加索引 |
| `ON DELETE CASCADE` vs `SET NULL` | 删用户 → 笔记一起删；删题目 → 笔记保留（笔记本身有价值） |
| 复习状态独立成 `review_cards` | 复习算法参数是算法实现细节，不应污染笔记业务表 |
| `review_logs` 只追加不修改 | 审计日志模式，可回溯复习曲线，便于做统计图表 |
| 向量单独成 `note_embeddings` 表 | 向量是派生数据，可重建；且避免主表被大字段拖慢 |
| `updated_at` 用触发器维护 | 避免每处代码都要手动赋值 |

```sql
-- 通用 updated_at 触发器
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_notes_updated_at
    BEFORE UPDATE ON notes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
```

### 5.6 SM-2 算法速览

```
输入：当前卡片的 (ease_factor, interval_days, repetitions)，用户评分 q ∈ [0,5]

if q < 3:                        # 答错，重来
    repetitions = 0
    interval_days = 1
else:
    if repetitions == 0: interval_days = 1
    elif repetitions == 1: interval_days = 6
    else: interval_days = round(interval_days * ease_factor)
    repetitions += 1

ease_factor = ease_factor + (0.1 - (5 - q) * (0.08 + (5 - q) * 0.02))
ease_factor = max(1.3, ease_factor)

due_at = now() + interval_days 天
```

---

## 6. API 设计

### 6.1 通用约定

- **Base URL**：`/api/v1`
- **鉴权**：`Authorization: Bearer <access_token>`
- **时间格式**：ISO 8601 UTC（`2026-09-23T08:30:00Z`）
- **命名**：URL 用复数名词 + kebab-case，JSON 字段用 snake_case
- **服务归属**：🟦 = `leetnote-api` (Go)　🐍 = `leetnote-ai` (Python)

**统一响应结构**

成功（分页）：

```json
{
  "items": [ { "id": 1, "title": "两数之和" } ],
  "total": 42,
  "page": 1,
  "size": 20,
  "pages": 3
}
```

失败（HTTP 状态码语义化 + 统一错误体）：

```json
{
  "error": {
    "code": "NOTE_NOT_FOUND",
    "message": "笔记不存在",
    "details": {}
  }
}
```

**状态码约定**

| 码 | 场景 |
|---|---|
| 200 | 查询/更新成功 |
| 201 | 创建成功 |
| 204 | 删除成功（无响应体） |
| 400 | 参数校验失败 |
| 401 | 未登录 / Token 失效 |
| 403 | 已登录但无权限（访问别人的笔记） |
| 404 | 资源不存在 |
| 409 | 冲突（用户名已存在） |
| 422 | 请求体校验失败 |
| 429 | 触发限流 |
| 500 | 服务端异常 |

**分页参数**：`?page=1&size=20`（`size` 上限 100）

### 6.2 接口清单

#### 认证 `/auth` 🟦

| 方法 | 路径 | 说明 | 鉴权 |
|---|---|---|---|
| POST | `/auth/register` | 注册 | ✗ |
| POST | `/auth/login` | 登录，返回 access + refresh | ✗ |
| POST | `/auth/refresh` | 用 refresh token 换新 access token | ✗ |
| POST | `/auth/logout` | 登出（吊销 refresh token） | ✓ |

#### 用户 `/users` 🟦

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/users/me` | 当前用户信息 |
| PATCH | `/users/me` | 修改昵称/头像/简介 |
| POST | `/users/me/password` | 修改密码 |

#### 题目 `/problems` 🟦

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/problems` | 列表，支持 `keyword` `difficulty` `tag_id` `sort` 筛选 + 分页 |
| GET | `/problems/{id}` | 详情 |
| POST | `/problems` | 手动新增题目 |
| PUT | `/problems/{id}` | 全量修改 |
| DELETE | `/problems/{id}` | 删除 |

`sort` 的取值走**白名单**（非法值返回 422，不会拼进 SQL）：

| 值 | 含义 | 实际 ORDER BY |
|---|---|---|
| `tidan`（默认） | 题单顺序，从易到难 | `sort_order = 0, sort_order, leetcode_id, id` |
| `leetcode` | 题号 | `leetcode_id NULLS LAST, id` |
| `topic` | 先按专题分组，组内按题单顺序 | 相关子查询取该题专题的最小 `sort_order` |
| `rating` / `rating_desc` | 难度分升 / 降序 | `rating [DESC] NULLS LAST, leetcode_id, id` |
| `title` | 标题 | `title, id` |

两个设计点：

- `sort_order = 0` 是布尔表达式，PostgreSQL 里 `false < true`，所以「非题单题目」
  （`sort_order = 0`）自然落到最后，不用额外写特例。
- 每个排序都补了 `id` 作为**最后的 tiebreaker**。缺了它，等值行在不同查询间的
  相对顺序不确定，翻页时会出现「第 1 页和第 2 页有重复行」。

#### 笔记 `/notes` 🟦

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/notes` | 列表，支持 `keyword` `difficulty` `tag` `status` `starred` `sort` + 分页 |
| GET | `/notes/{id}` | 详情（含解法、标签） |
| POST | `/notes` | 新建 |
| PUT | `/notes/{id}` | 全量修改（含解法与标签） |
| DELETE | `/notes/{id}` | 删除 |
| POST | `/notes/{id}/star` | 切换收藏 |

#### 解法 `/notes/{note_id}/solutions` 🟦

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/notes/{note_id}/solutions` | 该笔记的全部解法 |
| POST | `/notes/{note_id}/solutions` | 新增解法 |
| PUT | `/solutions/{id}` | 修改解法 |
| DELETE | `/solutions/{id}` | 删除解法 |

#### 标签 `/tags` 🟦

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/tags` | 全部标签，支持 `kind` 筛选 |
| POST | `/tags` | 新建标签 |
| DELETE | `/tags/{id}` | 删除标签 |

#### 统计 `/stats` 🟦

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/stats/overview` | 总数 / 按难度分布 / 连续打卡天数 |
| GET | `/stats/trend?days=30` | 近 N 天每日新增数 |

#### 搜索 `/search` 🟦

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/search?q=&type=note\|problem` | 全文检索（pg_trgm） |

#### 复习 `/reviews` 🟦

| 方法 | 路径 | 说明 | 状态 |
|---|---|---|---|
| GET | `/reviews/due?limit=20` | 当前到期待复习的卡片 | ✅ |
| POST | `/reviews/{card_id}/submit` | 提交自评 `{ "rating": 4 }`，按 SM-2 更新 | ✅ |
| GET | `/reviews/stats` | 复习统计 + 未来 7 天到期分布 | ✅ |

**SM-2 算法**（实现在 `internal/service/review_service.go` 的 `ApplySM2`，纯函数）：

```
rating < 3（没答上来）：连续次数归零，间隔重置为 1 天，但【难度系数保留】
rating ≥ 3（答对）：  间隔按 1 → 6 → 上次间隔 × 难度系数 增长
难度系数：EF += 0.1 - (5-q) × (0.08 + (5-q) × 0.02)，下限 1.3
```

> 「答错时保留难度系数」是容易被简化实现写错的地方：如果把 EF 也重置回 2.5，
> 一张反复答错的难题就永远不会因为「难」而缩短间隔，复习负担会越滚越大。

#### 前端可见的 AI 能力 🟦

> **前端不直接访问 AI 服务**。Go 侧统一鉴权后转发到 `leetnote-ai`，
> 这样用户认证只实现一次，AI 服务也不必暴露到公网。

| 方法 | 路径 | 说明 | 状态 |
|---|---|---|---|
| GET | `/notes/{id}/similar?limit=5` | 相似题推荐 | ✅ 已实现 |
| POST | `/notes/{id}/explain` | LLM 解法讲解（同步，约 10s） | ✅ 已实现 |

#### `leetnote-ai` 内部接口 🐍

> 需要请求头 `X-Internal-Token`，**不对前端暴露**。

| 方法 | 路径 | 说明 | 状态 |
|---|---|---|---|
| GET | `/health` | 健康检查（不校验令牌，供探活） | ✅ |
| POST | `/api/v1/ai/embed` | 为单篇笔记生成/重建向量 | ✅ |
| POST | `/api/v1/ai/embed/batch` | 批量补向量（换模型后重建） | ✅ |
| GET | `/api/v1/ai/embedding/status?user_id=` | 向量覆盖率 | ✅ |
| POST | `/api/v1/ai/similar` | 相似题检索 | ✅ |
| POST | `/api/v1/ai/explain` | LLM 生成解法讲解（OpenCode Go） | ✅ |
| POST | `/api/v1/ai/review-card` | LLM 生成复习卡 | 📋 待实现 |
| GET | `/api/v1/ai/graph/related` | 知识点图谱（GraphRAG） | 📋 待实现 |

#### 第三方登录 `/auth/github` 🟦

| 方法 | 路径 | 说明 | 鉴权 |
|---|---|---|---|
| GET | `/auth/github/authorize` | 返回授权地址（服务端生成 state） | ✗ |
| GET | `/auth/github/callback` | GitHub 授权后浏览器跳转至此（302 回前端） | ✗ |
| GET | `/users/me/oauth-accounts` | 查询已绑定的第三方账号 | ✓ |
| DELETE | `/users/me/oauth-accounts/github` | 解绑 GitHub（无密码账号禁止解绑） | ✓ |

**OAuth 登录流程**

```mermaid
sequenceDiagram
    participant U as 用户
    participant F as Vue 前端
    participant G as leetnote-api (Go)
    participant GH as GitHub

    U->>F: 点击「使用 GitHub 登录」
    F->>G: GET /auth/github/authorize
    G->>G: 生成随机 state，写入 httpOnly Cookie
    G-->>F: { authorize_url }
    F->>GH: 跳转到 authorize_url
    U->>GH: 授权
    GH->>G: 302 回调 ?code=xxx&state=xxx
    G->>G: 比对 state（一次性，恒定时间比较）
    G->>GH: POST /login/oauth/access_token
    GH-->>G: access_token
    G->>GH: GET /user + /user/emails
    GH-->>G: 用户信息（含已验证邮箱）
    G->>G: 已关联→登录 / 邮箱命中→关联 / 否则→建号
    G-->>F: 302 回前端 + 下发 refresh Cookie
    F->>G: POST /auth/refresh 换取 access_token
```

> **关键安全点**：
> - `state` 防 CSRF，且**一次性使用**（用过即焚，防重放）
> - 比对用 `subtle.ConstantTimeCompare`，避免时序侧信道
> - **只认 GitHub 已验证的邮箱**，否则攻击者填别人的邮箱就能接管账号
> - **token 绝不放进跳转 URL**（会进浏览器历史 / Referer / 服务器日志），改为下发 httpOnly Cookie
> - `redirect` 参数只接受站内相对路径，防开放重定向

### 6.3 gRPC 契约（📋 计划中，当前用 HTTP/JSON）

> 下面这份契约是**为将来的 gRPC 升级预留的**。当前实现走 HTTP/JSON，
> 接口语义与这份 proto 一致，将来替换时 service 层不用改。

`proto/leetnote.proto`：

```protobuf
syntax = "proto3";
package leetnote.v1;

option go_package = "github.com/mzzzmnq/leetnote-api/proto/leetnotev1";

service AiService {
  // Go 侧在创建/更新笔记后异步调用，重建向量
  rpc EmbedNote (EmbedNoteRequest) returns (EmbedNoteResponse);
  // Go 侧调用，获取相似题
  rpc FindSimilar (FindSimilarRequest) returns (FindSimilarResponse);
  // Go 侧调用，生成复习卡
  rpc GenerateReviewCard (GenerateReviewCardRequest) returns (GenerateReviewCardResponse);
}

message EmbedNoteRequest {
  int64  note_id = 1;
  string title   = 2;
  string content = 3;
}

message EmbedNoteResponse {
  bool   ok     = 1;
  string error  = 2;
}

message FindSimilarRequest {
  int64 note_id = 1;
  int32 limit   = 2;
}

message SimilarNote {
  int64   note_id    = 1;
  string  title      = 2;
  float   similarity = 3;
}

message FindSimilarResponse {
  repeated SimilarNote items = 1;
}

message GenerateReviewCardRequest {
  int64 note_id = 1;
}

message GenerateReviewCardResponse {
  string question = 1;
  string answer   = 2;
}
```

> **面试加分点**：能说清"为什么服务间用 gRPC 而不是 REST"——强类型契约、代码自动生成、二进制传输更省带宽、支持流式。

### 6.4 关键请求/响应示例

**POST `/api/v1/notes`**

```json
{
  "problem_id": 12,
  "title": "两数之和 · 哈希表解法",
  "content_md": "## 思路\n用哈希表记录已访问元素……",
  "summary": "一次遍历，用字典把查找降到 O(1)",
  "status": "published",
  "tag_ids": [3, 7],
  "solutions": [
    {
      "title": "哈希表 · 一次遍历",
      "language": "go",
      "code": "func twoSum(nums []int, target int) []int {\n    seen := map[int]int{}\n    ...",
      "time_complexity": "O(n)",
      "space_complexity": "O(n)",
      "sort_order": 0
    }
  ]
}
```

**201 Created**

```json
{
  "id": 101,
  "title": "两数之和 · 哈希表解法",
  "status": "published",
  "problem": { "id": 12, "leetcode_id": 1, "title": "Two Sum", "difficulty": "Easy" },
  "tags": [ { "id": 3, "name": "哈希表" }, { "id": 7, "name": "数组" } ],
  "solutions": [ { "id": 55, "title": "哈希表 · 一次遍历", "language": "go" } ],
  "created_at": "2026-09-23T08:30:00Z",
  "updated_at": "2026-09-23T08:30:00Z"
}
```

---

## 6.5 前端主题系统与无障碍

### 为什么用 CSS 变量而不是 Sass 变量

Sass 变量在编译期就被替换成字面量，运行时改不了；主题切换必须发生在运行时，
所以只能用 **CSS 自定义属性**（`--ln-*`）。

所有颜色集中定义在 `styles/main.css`：

```css
:root      { --ln-bg: #f6f7f9; --ln-text: #1a1f2b; --ln-primary: #2f6fed; ... }
html.dark  { --ln-bg: #0e1116; --ln-text: #e6edf3; --ln-primary: #6b9bff; ... }
```

换肤就是切一个类名：

```ts
document.documentElement.classList.toggle('dark', isDark)
```

组件里**零硬编码颜色**，所以新增页面自动同时支持亮暗两套，不需要额外适配。

### 不闪白（FOUC）

主题如果等 Vue 挂载后再应用，页面会先渲染一帧白底再变暗。所以把它放在
`index.html` 的**内联脚本**里，位置在样式表和模块脚本之前：

```html
<script>
  ;(function () {
    var saved = localStorage.getItem('leetnote:theme')
    var dark = saved === 'dark' ||
      ((!saved || saved === 'auto') && window.matchMedia('(prefers-color-scheme: dark)').matches)
    document.documentElement.classList.toggle('dark', dark)
  })()
</script>
```

这是**唯一**必须在框架之外写的一小段逻辑，因为它要跑得比框架早。

### 四处必须同步换肤

只改自己写的 CSS 是不够的，还有三个"别人家的"颜色系统：

| 系统 | 适配方式 |
|---|---|
| 自己的界面 | CSS 变量 |
| **Naive UI 组件** | `NConfigProvider` 的 `theme` + `themeOverrides` |
| **ECharts 图表** | `registerTheme` 注册两套主题；切主题时**销毁重建**实例（ECharts 只在 `init` 时读主题） |
| **代码高亮** | 手写 hljs 配色（官方主题是全局规则，没法按主题切换） |

任何一处漏掉，都会出现"页面变暗了但弹窗还是白的"这种割裂感。

### 无障碍：不是"锦上添花"

按 WCAG AA 逐元素实测对比度后，发现并修掉了这些问题：

| 位置 | 修前 | 修后 |
|---|---|---|
| 导航选中项 | 4.05:1 | 5.86:1（拆出 `--ln-primary-text`） |
| Naive 输入框占位符 | 1.78:1 | 4.83:1 |
| Naive 标签文字（warning） | 1.93:1 | 4.85:1 |
| Naive 空状态提示 | 1.78:1 | 6.0:1 |
| 头像白字压浅灰底 | 1.61:1 | 4.55:1 |
| 分页当前页 | 4.24:1 | 5.86:1 |
| 收藏星标 | 1.51:1 | 4.83:1 |

结论是 **Lighthouse 无障碍 1.0**（9 个页面 × 亮暗两套主题，零失败项）。

两个 Naive UI 的坑值得单独记：

- **`n-form-item` 的 `label` 没有 `for` 属性**，渲染出的 `<label>` 和输入框之间
  没有程序化关联，屏幕阅读器读不出名称 —— 必须自己补 `aria-label`。
- **`n-input` 会把未知属性透传到外层 `<div>`**，而不是内层 `<input>`。
  直接写 `aria-label` 不生效，要用 `:input-props="{ 'aria-label': '...' }"`。

### 主色为什么要拆成两个 token

```css
--ln-primary: #2f6fed;       /* 按钮底色、图表线条 */
--ln-primary-text: #1f56c4;  /* 链接、导航选中文字 */
```

同一个蓝色，当**文字**放在 9% 主色浅底上只有 4.05:1，不达标；加深后是 5.86:1。
而**按钮底色**不需要满足文字对比度，保持鲜亮更好看。两者的约束不一样，
所以不能共用一个值。

---

## 7. 项目目录结构

```
leetnote/
├── leetnote-api/                     # ★ Go 主服务
│   ├── cmd/
│   │   └── server/
│   │       └── main.go               # 入口：装配依赖、启动 HTTP + gRPC
│   ├── internal/
│   │   ├── config/                   # Viper 配置加载（.env → struct）
│   │   ├── router/                   # 路由注册
│   │   ├── handler/                  # HTTP 处理层（参数绑定、响应）
│   │   │   ├── auth.go
│   │   │   ├── note.go
│   │   │   ├── problem.go
│   │   │   ├── tag.go
│   │   │   ├── stats.go
│   │   │   └── review.go
│   │   ├── service/                  # 业务逻辑层
│   │   │   ├── auth_service.go
│   │   │   ├── note_service.go
│   │   │   ├── review_service.go     # SM-2 算法
│   │   │   └── stats_service.go
│   │   ├── repository/               # 数据访问层（pgx）
│   │   │   ├── user_repo.go
│   │   │   ├── note_repo.go
│   │   │   └── ...
│   │   ├── model/                    # 领域模型（对应表）
│   │   ├── dto/                      # 请求/响应结构体 + 校验标签
│   │   ├── middleware/               # JWT / 日志 / 恢复 / 限流 / CORS
│   │   ├── grpcclient/               # 调用 leetnote-ai 的 gRPC 客户端
│   │   └── pkg/
│   │       ├── jwt/
│   │       ├── hash/                 # bcrypt
│   │       ├── response/             # 统一响应封装
│   │       └── errs/                 # 业务错误定义
│   ├── migrations/                   # golang-migrate SQL 文件
│   │   ├── 000001_init.up.sql
│   │   └── 000001_init.down.sql
│   ├── tests/
│   ├── go.mod
│   ├── .env.example
│   ├── Makefile                      # run / test / lint / migrate
│   └── Dockerfile
│
├── leetnote-ai/                      # ★ Python AI 服务
│   ├── app/
│   │   ├── main.py                   # FastAPI 入口
│   │   ├── core/
│   │   │   ├── config.py             # Pydantic Settings
│   │   │   └── logging.py
│   │   ├── api/v1/
│   │   │   ├── explain.py            # LLM 解法讲解
│   │   │   ├── similar.py            # 相似题检索
│   │   │   ├── graph.py              # GraphRAG 知识点图谱
│   │   │   └── review_card.py
│   │   ├── rag/
│   │   │   ├── embedding.py          # 向量化
│   │   │   ├── retriever.py          # pgvector 检索
│   │   │   ├── rerank.py             # 重排
│   │   │   ├── graphrag.py           # 图谱检索
│   │   │   └── evaluate.py           # 检索质量评估
│   │   ├── llm/
│   │   │   ├── client.py             # LLM API 封装（超时/重试/降级）
│   │   │   └── prompts.py            # prompt 模板
│   │   ├── db/
│   │   │   └── session.py
│   │   ├── schemas/                  # Pydantic 模型
│   │   └── grpc_server.py            # 供 Go 调用
│   ├── tests/
│   ├── pyproject.toml
│   ├── .env.example
│   └── Dockerfile
│
├── frontend/                         # Vue 3 前端
│   ├── index.html                    # ★ 含防闪白的内联主题脚本
│   ├── public/robots.txt
│   ├── src/
│   │   ├── main.ts                   # 入口：Pinia → Router → 主题初始化
│   │   ├── App.vue                   # ★ 注入 Naive 的亮/暗主题
│   │   ├── api/
│   │   │   ├── client.ts             # axios 实例 + 401 单飞自动刷新
│   │   │   ├── token.ts              # access token 只存内存
│   │   │   ├── types.ts              # 与后端契约对应的 TS 类型
│   │   │   └── auth/notes/problems/tags/reviews/stats/search/users.ts
│   │   ├── router/index.ts           # 路由与登录守卫
│   │   ├── stores/
│   │   │   ├── user.ts
│   │   │   └── theme.ts              # ★ 主题状态（浅色/深色/跟随系统）
│   │   ├── layouts/
│   │   │   └── DefaultLayout.vue     # 顶栏 + 导航 + 主题切换
│   │   ├── views/
│   │   │   ├── DashboardView.vue     # 概览（统计卡 + 图表）
│   │   │   ├── NoteListView.vue
│   │   │   ├── NoteDetailView.vue
│   │   │   ├── NoteEditView.vue      # 分屏 Markdown 编辑器
│   │   │   ├── ProblemListView.vue   # 题目库（难度分 + 6 种排序）
│   │   │   ├── ReviewView.vue        # 闪卡式复习
│   │   │   ├── SearchView.vue
│   │   │   ├── SettingsView.vue
│   │   │   ├── LoginView.vue / RegisterView.vue / OAuthCallbackView.vue
│   │   │   └── NotFoundView.vue
│   │   ├── components/
│   │   │   ├── BaseChart.vue         # ECharts 封装（随主题重建）
│   │   │   ├── MarkdownViewer.vue    # Markdown + 代码高亮
│   │   │   ├── SolutionEditor.vue    # 多解法增删改
│   │   │   ├── NoteCard.vue
│   │   │   ├── DifficultyTag.vue
│   │   │   ├── RatingBadge.vue       # 难度分徽标
│   │   │   └── ThemeToggle.vue       # ★ 主题切换控件
│   │   ├── styles/
│   │   │   ├── main.css              # ★ 设计令牌（亮/暗两套）+ 全局样式
│   │   │   └── naive-theme.ts        # ★ Naive UI 主题覆盖
│   │   └── utils/
│   │       ├── chartTheme.ts         # ★ ECharts 亮/暗主题
│   │       ├── markdown.ts           # markdown-it + highlight.js 按需注册
│   │       ├── format.ts
│   │       └── query.ts
│   ├── vite.config.ts
│   └── tsconfig.json
│
├── proto/
│   └── leetnote.proto                # ★ 双服务共享的 gRPC 契约
│
├── deploy/
│   ├── docker-compose.yml            # 生产：4 服务编排
│   ├── docker-compose.dev.yml        # 本地：只起 postgres + redis
│   └── nginx.conf                    # 路径分流到 Go / Python
│
├── .github/workflows/
│   ├── api-ci.yml                    # Go: lint → test → build
│   ├── ai-ci.yml                     # Python: ruff → pytest
│   └── web-ci.yml                    # 前端: eslint → build
│
├── docs/
│   ├── DESIGN.md                     # 本文档
│   └── ROADMAP.md                    # 学习路线图
└── README.md
```

---

## 8. 认证方案

**双 Token 策略**：

| Token | 有效期 | 存放位置 | 作用 |
|---|---|---|---|
| access_token | 15 分钟 | 前端内存（Pinia） | 访问受保护接口 |
| refresh_token | 7 天 | httpOnly + Secure Cookie | 换取新 access_token |

**流程**：

```mermaid
sequenceDiagram
    participant U as 用户
    participant F as Vue 前端
    participant G as leetnote-api (Go)
    participant D as PostgreSQL

    U->>F: 输入账号密码
    F->>G: POST /auth/login
    G->>D: 查用户 + bcrypt 校验
    D-->>G: 用户记录
    G-->>F: access_token(JSON) + refresh_token(Cookie)
    F->>F: access_token 存 Pinia 内存

    U->>F: 打开笔记列表
    F->>G: GET /notes (Bearer access)
    G-->>F: 200 数据

    Note over F,G: 15 分钟后 access 过期
    F->>G: GET /notes → 401
    F->>G: POST /auth/refresh (Cookie 自动携带)
    G-->>F: 新 access_token
    F->>G: 重放原请求 → 200
```

**Go 侧实现要点**：
- `middleware.Auth()` 解析 Bearer Token，把 `user_id` 注入 `gin.Context`
- `handler` 层通过 `c.Get("user_id")` 取当前用户，**所有查询强制带 `user_id` 条件**（防越权）
- refresh_token 的 `jti` 写入 Redis 黑名单实现登出（M8）

**为什么 refresh_token 放 httpOnly Cookie？** 防 XSS 窃取；access_token 放内存、不放 localStorage，同理。代价是要处理 CSRF——refresh 接口需校验 `SameSite=Lax`。

---

## 9. 部署方案（零服务器起步）

先用**免费托管平台**跑通并写进简历，后期再迁移到云服务器。

| 组件 | 免费方案 | 说明 |
|---|---|---|
| 前端 | **Vercel** / Cloudflare Pages | 自动 HTTPS、全球 CDN |
| Go 服务 | **Zeabur** / Render / Fly.io | Render 免费层 15 分钟休眠，冷启动 30–50s |
| Python 服务 | 同上 | 可与 Go 服务放同一平台 |
| 数据库 | **Neon** / Supabase | Serverless PostgreSQL，支持 pgvector |
| Redis | Upstash | 免费 10K 命令/天 |
| 图片存储 | Cloudflare R2 | 免费 10 GB |
| CI/CD | GitHub Actions | 免费 2000 分钟/月 |

> **国内访问提醒**：Vercel / Render 在国内可能较慢。面试演示建议备选 **Zeabur**（新加坡节点），或后期买 2C2G 云服务器（学生机约 ¥10/月）自建。

### 环境变量清单

```dotenv
# ---- leetnote-api/.env (Go) ----
APP_ENV=development
HTTP_PORT=8080
GRPC_PORT=9090
DATABASE_URL=postgres://leetnote:leetnote@localhost:5432/leetnote?sslmode=disable
REDIS_ADDR=localhost:6379
JWT_SECRET=<用 openssl rand -hex 32 生成>
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=168h
AI_GRPC_ADDR=localhost:9090
CORS_ORIGINS=http://localhost:5173

# ---- leetnote-ai/.env (Python) ----
APP_ENV=development
HTTP_PORT=8000
GRPC_PORT=9090
DATABASE_URL=postgresql+psycopg://leetnote:leetnote@localhost:5432/leetnote
LLM_PROVIDER=openai
LLM_API_KEY=sk-xxx
LLM_BASE_URL=https://api.openai.com/v1
EMBEDDING_MODEL=text-embedding-3-small
EMBEDDING_DIM=1536

# ---- frontend/.env ----
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

### 本地一键启动

```yaml
# deploy/docker-compose.dev.yml
services:
  db:
    image: pgvector/pgvector:pg18        # 自带 pgvector 扩展
    environment:
      POSTGRES_USER: leetnote
      POSTGRES_PASSWORD: leetnote
      POSTGRES_DB: leetnote
    ports: ["5432:5432"]
    volumes: ["pgdata:/var/lib/postgresql/data"]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U leetnote"]
      interval: 5s
      retries: 10

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

volumes:
  pgdata:
```

```bash
docker compose -f deploy/docker-compose.dev.yml up -d   # 只起数据库

cd leetnote-api && make migrate-up && make run          # Go    :8080
cd leetnote-ai  && uvicorn app.main:app --reload        # Python :8000
cd frontend     && npm run dev                          # Vue   :5173
```

> 没有 Docker 时，也可以用**便携版 PostgreSQL**（见 `docs/ENVIRONMENT.md`）本地起库。

---

## 10. 学习路线

**详见 [`ROADMAP.md`](./ROADMAP.md)**（12 个月逐周清单）。

一句话版本：

```
阶段一 2026.10–12  地基      算法 180 题 · 计网/OS · Go 语言 · 项目 M1–M3
阶段二 2027.01–03  Go 工程   数据库/Redis/Docker · 项目上线 M6
阶段三 2027.04–06  AI 专项   RAG + GraphRAG · 双服务 M7–M8
阶段四 2027.07–09  冲刺      八股 · 简历 · 投递 · 面试
```

**学习方法**：
1. **项目驱动**——每学一个知识点立刻用到项目里
2. **官方文档优先**——Go / PostgreSQL / Gin 文档质量极高
3. **费曼技巧**——每个知识点用 LeetNote 自己写一篇笔记讲清楚
4. **先跑通再优化**——先写 N+1 查询，再用预加载优化，记录"320ms → 45ms"
5. **每天提交**——GitHub 的绿色方块是学习态度的证明

---

## 11. 简历怎么写

### 项目描述模板

> **LeetNote · 算法练习笔记系统**（个人项目）　2026.10 – 2027.07
> 技术栈：Go · Gin · PostgreSQL · pgvector · Python · FastAPI · gRPC · Vue 3 · TypeScript · Docker · GitHub Actions

- 采用 **Go + Python 双服务架构**：Go(Gin) 承载业务 API 并**统一负责用户鉴权**，Python(FastAPI) 负责向量化与相似度检索；服务间通过 HTTP + 内部令牌通信，**鉴权只在 Go 侧做一次**（AI 服务不重复实现用户认证）
- 基于 PostgreSQL 设计 **9 张表**的关系模型，通过**部分唯一索引**与**外键级联策略**保证一致性；使用 **golang-migrate** 管理表结构演进。
- 针对中文笔记检索，采用 **`pg_trgm` + GIN 索引**替代 PostgreSQL 原生全文检索（中文分词受限），将关键字查询从 **820ms 降至 90ms**。
- 使用 **pgvector + HNSW 索引**实现相似题推荐，构建检索评估集（召回率/命中率）迭代优化，Top-5 命中率达 **XX%**。
- 实现基于 **SM-2 算法**的间隔重复复习系统，自动计算复习间隔，支持"今日待复习"队列。
- 前端基于 **Vue 3 + TypeScript + Pinia**，封装 Axios 拦截器完成 **JWT 双 Token 自动续期**。
- 使用 **Docker Compose** 统一本地环境，通过 **GitHub Actions** 实现 lint → test → build → deploy 流水线，Go 侧用 golangci-lint 保证代码质量。

### 写简历的几条铁律

1. **每条都要有动词 + 技术 + 结果**。不要写"负责后端开发"，要写"用 Go 实现 X，解决 Y，达到 Z"。
2. **能量化就量化**：接口数、表数、覆盖率、响应时间、命中率。没有真实数据就自己压测出来。
3. **只写你真的能讲 30 分钟的**。面试官会顺着简历问到底。
4. **准备 3 个"踩坑故事"**，例如：
   - N+1 查询导致列表页 3 秒 → 用预加载优化到 200ms → 加测试防回归
   - 中文全文检索效果差 → 分析后改用 pg_trgm → 响应时间降 90%
   - Go 与 Python 服务间超时 → 引入 context 超时 + 重试 + 熔断

---

## 12. 面试自测题

**Go**
- GMP 调度模型是什么？为什么 Go 的 goroutine 比线程轻量？
- GC 的三色标记法怎么工作？STW 发生在哪些阶段？
- 什么是内存逃逸？怎么用 `go build -gcflags=-m` 排查？
- `slice` 和 `array` 的区别？slice 扩容策略是什么？
- channel 的底层实现？有缓冲和无缓冲的区别？
- `context` 的用途？怎么实现超时取消？
- interface 的底层结构（iface / eface）？nil interface 和 nil 指针的区别？

**数据库**
- 为什么 `problems` 表要独立出来？
- 部分唯一索引 `WHERE problem_id IS NOT NULL` 解决了什么问题？
- B+ 树索引为什么适合范围查询？什么情况下索引会失效？
- 联合索引的最左前缀原则？覆盖索引是什么？
- 事务的四个隔离级别分别解决什么问题？PostgreSQL 默认是哪个？MVCC 怎么实现？
- HNSW 索引和 IVFFlat 索引的区别？向量检索为什么用近似算法？

**后端与架构**
- JWT 无状态，"登出"怎么实现？
- 什么是 N+1 查询？Go 里怎么避免？
- 为什么服务间用 gRPC 而不是 REST？
- 分页 `LIMIT 20 OFFSET 100000` 为什么慢？游标分页怎么优化？
- 缓存穿透/击穿/雪崩分别是什么？怎么解决？
- 分布式锁怎么实现？Redis 方案有什么坑？

**AI 工程**
- RAG 的完整链路是什么？每一步可能出什么问题？
- 怎么评估检索质量？召回率和命中率的区别？
- chunking 策略有哪些？怎么选？
- rerank 为什么能提升效果？
- GraphRAG 相比普通 RAG 的优势是什么？
- 怎么防止 prompt 注入？

**前端**
- Vue3 的 `ref` 和 `reactive` 有什么区别？
- 路由守卫怎么实现"未登录跳转登录页"？
- Axios 拦截器如何实现 401 自动刷新 Token 并重放请求？

**工程化**
- Docker 镜像分层是什么？多阶段构建怎么写？
- CI/CD 里做了哪些检查？为什么测试要放在构建之前？

---

## 附：下一步

1. **环境搭建**：见 `docs/ENVIRONMENT.md`（Go / PostgreSQL 便携版安装）
2. **M1**：`leetnote-api` Go 骨架 + 配置 + 日志 + 健康检查
3. **M2**：认证模块 + JWT 中间件 + 单元测试
4. **数据库**：用 `migrations/000001_init.up.sql` 建表并验证

> 建议把每次踩坑、每个设计决策都记录到 LeetNote 自己身上——**边做边记录，项目完成时笔记也完成了**。

# 开发环境说明

> 本文档记录本机已配置的开发环境、安装位置、常用命令与待办事项。
> 配置日期：2026-09-23

---

## 1. 安装位置总览

| 组件 | 版本 | 安装位置 | 安装方式 |
|---|---|---|---|
| Go | **1.27.1** | `D:\dev\go` | 便携版 zip（免管理员） |
| PostgreSQL | **18.6** | `D:\dev\pgsql` | 便携版 binaries zip（免管理员） |
| Node.js | 已有 | `C:\Program Files\nodejs` | 系统已装 |
| Python | 已有（Anaconda） | `D:\Anaconda` | 系统已装 |
| Git | 已有 | `D:\Git` | 系统已装 |
| Docker | ❌ 未装 | — | 阶段二 W20 再装 |

**所有新增工具都在 `D:\dev\` 下，卸载只需删除文件夹。**

```
D:\dev\
├── go\                     # Go 工具链（GOROOT）
├── gopath\                 # Go 工作区（GOPATH）
│   ├── bin\                # go install 的可执行文件
│   └── src\
├── pgsql\                  # PostgreSQL
│   ├── bin\                # 可执行文件（39 个）
│   ├── data\               # 数据库集群（PGDATA）
│   ├── server.log          # 服务日志
│   └── ...
├── _downloads\             # 安装包缓存（可删，省 440MB）
├── pg-start.ps1            # 启动数据库
├── pg-stop.ps1             # 停止数据库
└── pg-status.ps1           # 查看状态
```

---

## 2. 环境变量（已写入用户级，无需管理员）

| 变量 | 值 | 用途 |
|---|---|---|
| `GOROOT` | `D:\dev\go` | Go 安装目录 |
| `GOPATH` | `D:\dev\gopath` | Go 工作区 |
| `GOPROXY` | `https://goproxy.cn,direct` | **国内代理**，否则拉依赖会超时 |
| `PGDATA` | `D:\dev\pgsql\data` | 数据库数据目录 |
| `PGHOST` | `localhost` | |
| `PGPORT` | `5432` | |
| `Path` | 追加了 `D:\dev\go\bin`、`D:\dev\gopath\bin`、`D:\dev\pgsql\bin` | |

> ⚠️ **环境变量对新开的终端才生效。** 现有的 PowerShell / VS Code 窗口需要**重启**，否则 `go`、`psql` 命令找不到。

**验证方式**（重启终端后）：
```powershell
go version          # go version go1.27.1 windows/amd64
psql --version      # psql (PostgreSQL) 18.6
```

---

## 3. PostgreSQL

### 启动 / 停止

```powershell
D:\dev\pg-start.ps1     # 启动
D:\dev\pg-stop.ps1      # 停止
D:\dev\pg-status.ps1    # 查看状态
```

> 便携版**不会开机自启**，每次重启电脑后需要手动跑一次 `pg-start.ps1`。

### 连接信息

| 项 | 值 |
|---|---|
| 主机 / 端口 | `localhost` / `5432` |
| 超级用户 | `postgres` / `postgres` |
| 业务用户 | `leetnote` / `leetnote` |
| 数据库 | `leetnote` |
| 连接串 | `postgres://leetnote:leetnote@localhost:5432/leetnote?sslmode=disable` |

### 常用命令

```powershell
# 进入交互式 SQL
psql -U leetnote -h localhost -d leetnote

# 查看所有表
psql -U leetnote -h localhost -d leetnote -c "\dt"

# 查看表结构
psql -U leetnote -h localhost -d leetnote -c "\d+ notes"

# 执行迁移文件
psql -U leetnote -h localhost -d leetnote -f migrations\000001_init.up.sql

# 重置数据库（清空重建）
psql -U postgres -h localhost -c "DROP DATABASE leetnote;"
psql -U postgres -h localhost -c "CREATE DATABASE leetnote OWNER leetnote;"
psql -U leetnote -h localhost -d leetnote -f migrations\000001_init.up.sql
```

> Windows 终端里 `psql` 的中文提示可能显示为乱码，那是**终端编码问题**，不影响功能。
> 想看清中文可以执行：`chcp 65001`（切到 UTF-8 代码页）。

### 已创建的表（8 张）

```
users · problems · tags · notes · solutions · note_tags · review_cards · review_logs
```

对应迁移文件：`leetnote-api/migrations/000001_init.up.sql`

### 已启用的扩展

- ✅ `pg_trgm` —— 中文子串检索（GIN 三元组索引）
- ✅ `pgcrypto` —— `gen_random_uuid()`
- ✅ `uuid-ossp`
- ❌ `vector`（pgvector）—— **待安装**，见下节

---

## 4. 待办：安装 pgvector

**为什么需要**：阶段三（2027.04）的相似题推荐依赖向量检索。

**现状**：EDB 便携版 binaries **不包含 pgvector**，需要单独处理。三种方案：

| 方案 | 难度 | 说明 |
|---|---|---|
| **A. 装 Docker，用 `pgvector/pgvector:pg18` 镜像** | 中 | **推荐**。顺便完成阶段二 W20 的 Docker 学习，且开发环境与生产一致 |
| B. 下载 pgvector 的 Windows 预编译包 | 中 | 需匹配 PostgreSQL 18 + MSVC 编译，找包麻烦 |
| C. 源码编译 | 高 | 需装 Visual Studio + C++ 工具链，不划算 |
| D. 用云数据库（Neon / Supabase） | 低 | 自带 pgvector，但要联网 |

**建议**：到阶段二装 Docker 时一并解决，用方案 A。

**在此之前**：不要执行 `migrations/000002_pgvector.up.sql`，会报 `extension "vector" is not available`。

---

## 5. Go 配置要点

### 已设置

```
GOROOT  = D:\dev\go
GOPATH  = D:\dev\gopath
GOPROXY = https://goproxy.cn,direct
```

> `GOPROXY` 指向国内代理（七牛云），**这是国内开发必须设置的**，否则 `go get` 大概率超时。

### 可选：换用更快的校验和数据库

如果 `go get` 报校验和错误，可以再加：
```powershell
go env -w GOSUMDB=sum.golang.google.cn
```

### 验证工具链

```powershell
go version                    # go1.27.1
go env GOPROXY                # https://goproxy.cn,direct
go run main.go                # 能跑通即可
```

---

## 6. 环境验证清单

配置完成后，逐条勾选：

- [x] `go version` 输出 `go1.27.1 windows/amd64`
- [x] `psql --version` 输出 `18.6`
- [x] `D:\dev\pg-status.ps1` 显示 `accepting connections`
- [x] `leetnote` 库存在，含 8 张表
- [x] `pg_trgm` 扩展已启用
- [x] Go 程序能连上 PostgreSQL 并查询（已用 pgx 实测通过）
- [ ] **重启终端后重新验证 `go` 和 `psql` 命令可用**
- [ ] 安装 DBeaver（可选，图形化看表更方便）

---

## 7. GitHub 访问与代理（重要）

国内直连 GitHub 会**连接被重置**（`Recv failure: Connection was reset`），必须走代理。

本机已用 **Clash Verge**（端口 `7897`）解决：

```powershell
# 只对 github.com 生效，不影响 gitee 等国内仓库
git config --global http.https://github.com.proxy http://127.0.0.1:7897
```

**排查口诀**：先看代理端口有没有在监听，再对比直连与走代理的差异：

```powershell
# 1. 代理端口在不在
Get-NetTCPConnection -LocalPort 7897 -State Listen

# 2. 直连（预期 000 = 失败）
curl.exe -s -o NUL -w "%{http_code}`n" --max-time 10 https://github.com

# 3. 走代理（预期 200 = 成功）
curl.exe -s -o NUL -w "%{http_code}`n" --max-time 15 --proxy http://127.0.0.1:7897 https://github.com
```

**注意事项**：

- Clash 端口随版本变化（Clash Verge 新版是 `7897`，老版 Clash for Windows 是 `7890`），换版本后要同步改 git 配置
- **系统代理开关关着不影响 git**——git 用的是自己配置的代理，两者独立
- 代理软件没开时 git 会失败，可临时取消：`git config --global --unset http.https://github.com.proxy`
- 同理，`go get` 拉取托管在 GitHub 上的依赖也依赖代理；已配的 `GOPROXY=goproxy.cn` 会先兜住大部分情况

---

## 8. 远程仓库

| 项 | 值 |
|---|---|
| 地址 | https://github.com/mzzzmnq/leetnote |
| 可见性 | 公开 |
| 默认分支 | `main` |
| 凭据 | Git Credential Manager 管理，已缓存 |

```powershell
cd D:\vibecoding_test\leetnote
git add .
git commit -m "feat: xxx"
git push
```

---

## 9. 下一步

环境已就绪，**M1（`leetnote-api` Go 骨架）已完成**：

```
leetnote-api/
├── cmd/server/main.go           入口：装配依赖 + 优雅关闭
├── internal/config/             环境变量 → 结构体
├── internal/logging/            slog 结构化日志
├── internal/db/                 pgx 连接池
├── internal/router/             路由 + 中间件装配
├── internal/handler/health.go   健康检查
├── internal/middleware/         RequestID / Logger / Recovery / CORS
├── internal/pkg/errs/           业务错误类型
├── internal/pkg/response/       统一响应封装
├── migrations/                  SQL 迁移
└── dev.ps1 / Makefile           开发任务
```

验收结果：`/health` 返回 `{"status":"ok","database":"ok"}`，
测试覆盖率 router 96.7% / errs 93.3% / response 93.3%。

**下一步 M2**：认证模块（bcrypt + JWT 双 Token + 鉴权中间件 + 4 个接口）。

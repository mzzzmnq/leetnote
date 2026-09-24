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

## 7. GitHub 访问（重要）

国内直连 GitHub 会被阻断。本机**已配置 SSH over 443**，日常 git 操作**不需要代理**。

### 7.1 首选方案：SSH over 443

**原理**：阻断发生在 SNI/DPI 层（实测：DNS 正常、用真实 IP 直连也失败，所以改 hosts 没用）。
GitHub 官方提供 `ssh.github.com:443` 作为备用入口——**SSH 协议不带 SNI 域名**，DPI 认不出目标，就放行了。

**`~/.ssh/config`**（原文件备份在 `config.bak`）：

```sshconfig
Host github.com
    HostName ssh.github.com     # 关键：走 443 入口
    Port 443
    User git
    PreferredAuthentications publickey
    IdentityFile ~/.ssh/id_ed25519_github
    IdentitiesOnly yes
    ServerAliveInterval 30
    ServerAliveCountMax 3
```

**两个必须知道的坑**：

**坑 1：Git 自带的 ssh 读不到配置**

Git for Windows 用的是 `D:\Git\usr\bin\ssh.exe`（MSYS2 版），它依赖 `$HOME` 找
`~/.ssh/config`。本机 `HOME` 为空，于是它**忽略配置、直连 22 端口**：

```
ssh: connect to host github.com port 22: Connection timed out
```

解法是让 git 显式使用 Windows 自带的 OpenSSH：

```powershell
git config --global core.sshCommand "C:/Windows/System32/OpenSSH/ssh.exe"
```

**坑 2：私钥不能有 passphrase**

如果私钥有 passphrase 而 `ssh-agent` 没运行，SSH 解不开私钥，
会报 `Permission denied (publickey)`——**看起来像密钥没注册，实际是加载不了**。

本机的 `id_ed25519_github` 是**无 passphrase** 的 ed25519 密钥，专供本机使用。

> 为什么不修好带口令的旧密钥：启动 `ssh-agent` 要管理员权限，且 Windows 的 agent
> **不跨重启保存密钥**，每次开机都得 `ssh-add`。无口令密钥才能真正"永久无感"。
> 代价是机器被入侵时密钥可被直接使用——缓解方式是这把密钥只用于 GitHub，可随时在网页删除。

**排查命令**：

```powershell
# 认证测试（成功会输出 Hi <用户名>! You've successfully authenticated）
ssh -T git@github.com

# 看 ssh 实际用了哪个配置（应显示 hostname ssh.github.com / port 443）
ssh -G github.com | Select-String 'hostname|port|identityfile'

# 对比：git 自带 ssh 是否忽略配置（若显示 port 22 说明踩到坑 1）
D:\Git\usr\bin\ssh.exe -G github.com | Select-String 'hostname|port'

# 查看远程分支（验证读权限）
git ls-remote --heads origin
```

> `ssh.github.com:443` 会**间歇性被 reset**，重试即可通过。

### 7.2 回退方案：HTTPS + 代理

SSH 通道万一长期不可用，切回 HTTPS 并挂上 Clash：

```powershell
# 切回 HTTPS
git remote set-url origin https://github.com/mzzzmnq/leetnote.git

# 只对 github.com 生效，不影响 gitee 等国内仓库
git config --global http.https://github.com.proxy http://127.0.0.1:7897
```

**注意事项**：

- Clash 端口随版本变化（Clash Verge 新版是 `7897`，老版 Clash for Windows 是 `7890`）
- **系统代理开关关着不影响 git**——git 用的是自己配置的代理，两者独立
- `go get` 拉取托管在 GitHub 的依赖也受此影响，但 `GOPROXY=goproxy.cn` 会先兜住大部分情况

### 7.3 限制说明

SSH over 443 **只解决 git 操作**（clone / pull / push）。
浏览器访问 github.com（查文档、提 issue、用 Actions 界面）**仍然需要代理**。

---

## 8. 远程仓库

| 项 | 值 |
|---|---|
| 地址 | `git@github.com:mzzzmnq/leetnote.git`（SSH） |
| 网页 | https://github.com/mzzzmnq/leetnote |
| 可见性 | 公开 |
| 默认分支 | `main` |
| 认证方式 | SSH 密钥 `~/.ssh/id_ed25519_github`（无口令） |
| 代理依赖 | **无** |

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

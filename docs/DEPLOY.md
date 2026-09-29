# 部署到另一台电脑

这份文档面向的场景是：**把这套项目完整搬到另一台机器上，带着数据一起。**

---

## 0. 先选一条路

| | 原生（PowerShell） | Docker |
|---|---|---|
| 机器上要装 | Go · Node · Python · PostgreSQL+pgvector | **只要 Docker Desktop** |
| 首次准备时间 | 1～3 小时（PostgreSQL + pgvector 最折腾） | 20 分钟 |
| 开发体验 | ✅ 热重载正常 | ⚠️ 文件监听在 Windows 上常失灵 |
| 资源占用 | 低 | Docker Desktop 常驻 2～4 GB |
| 适合 | **日常写代码** | **部署、以后上云** |

> **推荐组合**：平时用**原生**写代码，需要"一条命令起全栈"或上云时用 **Docker**。
> 两者共用同一份代码和同一套迁移，互不冲突。

两条路都往下看，选你要的那条。

---

## 1. 通用第一步：拿代码

```powershell
git clone git@github.com:mzzzmnq/leetnote.git
cd leetnote
```

> ⚠️ 仓库里**没有**这些（都被 .gitignore 排除了，需要在新机器重新生成）：
> `.env` × 3、`.venv/`、`node_modules/`、`bin/`、`local.config.ps1`。
> 下面两条路都会各自把缺的补上。

---

## 2. 路线一：原生（Windows）

### 2.1 装前置工具

| 工具 | 版本 | 安装方式 |
|---|---|---|
| **Go** | ≥ 1.24 | [go.dev/dl](https://go.dev/dl/) —— 便携版 zip 解压到 `D:\dev\go` 也行 |
| **Node.js** | ≥ 20 | [nodejs.org](https://nodejs.org/) LTS 安装包 |
| **Python** | 3.11 | [python.org](https://www.python.org/downloads/) —— **不要用 Anaconda**，见下方说明 |
| **PostgreSQL** | ≥ 16 | 见下一节（这是最麻烦的一步） |

> **为什么不能用 Anaconda 的 Python**
> 它自带的 OpenSSL 版本不兼容，会让 pip 的所有 HTTPS 请求失败，
> 表现是「pip 找不到任何包」，很容易误以为是网络问题。
> 用 python.org 的官方安装包就没有这个问题。

#### PostgreSQL + pgvector

官方**不提供** Windows 二进制，pgvector 得自己折腾。三种办法，按省事程度排序：

**① 用 Docker 只起数据库**（最省事，推荐）

```powershell
cd deploy
docker compose -f docker-compose.dev.yml up -d
```

得到的库连接串是 `postgres://leetnote:leetnote@localhost:5432/leetnote?sslmode=disable`，
和本机装的一模一样，其余步骤完全不变。

**② 用 EDB 的 PostgreSQL 安装包 + 自己编译 pgvector**

见 `docs/ENVIRONMENT.md` 的「pgvector 安装」一节 —— 本机就是这么装的，
里面有完整的踩坑记录。**需要 MSVC 编译环境**。

**③ 用 WSL2 里的 PostgreSQL**

能用，但跨文件系统访问会让数据库 IO 变慢，不太推荐。

### 2.2 一条命令初始化

```powershell
.\setup.ps1
```

它会自动做完：

| 步骤 | 说明 |
|---|---|
| ① 检查工具 | 找不到会明确告诉你缺哪个 |
| ② 生成 3 个 `.env` | **密钥每次都是随机生成的**（不是复制旧机器的） |
| ③ `npm install` | 前端依赖 |
| ④ 建 `.venv` + 装依赖 | 用 `requirements.txt` 装 AI 服务的依赖 |
| ⑤ 建库 + 装扩展 + 跑迁移 | 表结构由版本化迁移生成 |
| ⑥ 编译 Go 服务 | 产出 `bin\leetnote-api.exe` |

**幂等**：已经做过的步骤会自动跳过，可以反复跑。
只想检查环境不动任何东西：

```powershell
.\setup.ps1 -CheckOnly
```

### 2.3 搬数据（可选）

如果你要的是「新机器上继续用原来的笔记」：

1. **在旧机器上**导出：

   ```powershell
   cd D:\vibecoding_test\leetnote
   .\migrate-data.ps1 export
   # 产出 backups\leetnote-20260929-120000.dump（约 60 KB）
   ```

2. 把这个 `.dump` 文件**拷到新机器**（U 盘 / 网盘 / scp 都行）

3. **在新机器上**导入：

   ```powershell
   .\migrate-data.ps1 import -File .\leetnote-20260929-120000.dump
   ```

导入做的事：建库 → 装扩展 → 用迁移建表 → 只把数据灌进去。
好处是**顺手验证了新机器上的迁移是对的**。

> 想先看看 dump 里有什么、确认不是拷错了文件：
> ```powershell
> .\migrate-data.ps1 info -File .\xxx.dump
> ```

### 2.4 启动

```powershell
.\start-all.ps1
```

或者双击 **`LeetNote.bat`** 打开菜单。

浏览器打开 <http://localhost:5173>，用你原来的账号登录。

---

## 3. 路线二：Docker

### 3.1 装 Docker Desktop

[docker.com/products/docker-desktop](https://www.docker.com/products/docker-desktop/)

Windows 上它会启用 WSL2 后端 —— 也就是说 Docker 容器实际跑在一个轻量 Linux 虚拟机里。
（Linux 容器和 Windows 内核不兼容，这是必须的，不是 Docker 的缺陷。）

装完确认：

```powershell
docker --version
docker compose version
```

### 3.2 起全栈

```powershell
cd deploy
Copy-Item .env.example .env
```

**打开 `.env` 改三处**（这三个是必须改的，其他可留默认）：

```
DB_PASSWORD=自己起一个
JWT_SECRET=自己起一个（至少 32 位；生成方式见文件内注释）
AI_INTERNAL_TOKEN=自己起一个
```

然后：

```powershell
docker compose up -d --build
```

第一次会构建镜像（几分钟）。起来之后浏览器打开 <http://localhost>。

**整个过程不需要在本机装 Go / Python / Node / PostgreSQL。**

### 3.3 架构长什么样

```
                 ┌──────────────────────────────────────┐
   浏览器 ──────► │  web (nginx:80)                      │  ← 唯一暴露到宿主机的端口
   :80           │    ├─ 静态文件 → dist/                │
                 │    └─ /api/*  → 反向代理 ─┐           │
                 └───────────────────────────┼──────────┘
                                             ▼
                 ┌──────────────────────────────────────┐
                 │  api (Go:8080)                       │  ← 只在容器网络内可达
                 │    └─ AI_SERVICE_URL=http://ai:8000 ─┼──┐
                 └──────────────────────────────────────┘  │
                                             ┌─────────────▼────────────────────┐
                                             │  ai (Python:8000)                │
                                             └─────────────┬────────────────────┘
                                                           ▼
                                             ┌──────────────────────────────────┐
                                             │  db (postgres+pgvector:5432)     │
                                             │    volume: pgdata                │  ← 数据在这里
                                             └──────────────────────────────────┘

   migrate 是一次性任务：建表完成后退出，api 才启动
```

两个关键点：

- **只有 `web` 暴露端口**。数据库和 API 都在容器网络内，宿主机和局域网都碰不到。
- **`/api` 走 nginx 同源代理**。所以不需要 CORS，也不会有跨域 cookie 的问题。

### 3.4 搬数据

生产用的 compose **刻意不把 5432 暴露到宿主机**，所以不能用本机的 psql 直连。
在容器里还原即可：

```powershell
cd deploy

# 1. 把 dump 拷进数据库容器
docker compose cp ..\backups\leetnote-20260929-120000.dump db:/tmp/restore.dump

# 2. 确认表已经建好（第一次 up 时 migrate 服务会建）
docker compose logs migrate

# 3. 只还原数据（表结构已经由迁移建好）
docker compose exec db pg_restore --data-only --disable-triggers `
    -U leetnote -d leetnote /tmp/restore.dump
```

> `--disable-triggers` 会临时关掉外键约束，避免表之间的插入顺序导致失败。
> 它需要超级用户 —— 容器里的 `leetnote` 就是初始化时创建的超级用户，所以没问题。

还原完刷新浏览器就能看到数据了。

### 3.5 常用命令

```powershell
docker compose ps                    # 看状态
docker compose logs -f api           # 跟日志
docker compose exec api sh           # 进容器（alpine 有 shell）
docker compose down                  # 停止（保留数据）
docker compose down -v               # ⚠️ 连数据库卷一起删，数据全没
docker compose up -d --build         # 改代码后重新构建
```

---

## 4. 换机器检查清单

搬完之后挨个确认：

- [ ] 四个服务都在跑：`.\status.ps1`（原生）或 `docker compose ps`（Docker）
- [ ] 能打开 <http://localhost:5173>（原生）或 <http://localhost>（Docker）
- [ ] 能用**原来的账号**登录
- [ ] 登录后**刷新页面不会掉登录**
      → 掉了就是 cookie 的 `Secure` 属性问题，把 `COOKIE_SECURE` 设成 `false`
- [ ] 笔记列表里有你原来的笔记
- [ ] 题目库能看到 174 道题、难度分、6 种排序
- [ ] 笔记详情能切换语言标签页
- [ ] 复习页显示到期卡片

---

## 5. 排错

| 现象 | 原因 | 怎么办 |
|---|---|---|
| `pip` 报「找不到任何包」 | 用了 Anaconda 的 Python | 换 python.org 的官方版本重建 venv |
| `pip` 读 requirements.txt 报 `UnicodeDecodeError: 'gbk'` | pip 用系统区域编码读文件 | 文件已改成纯 ASCII；自己加注释时也**别写中文** |
| 登录成功但刷新就掉登录 | cookie 带了 `Secure`，但你在用 HTTP | 设 `COOKIE_SECURE=false` |
| 容器里连不上数据库 | 容器里的 `localhost` 是容器自己 | 连接串里写服务名 `db` 而不是 `localhost` |
| `docker compose down` 后数据没了 | 用了 `-v`，卷被删了 | 数据在 volume 里，`down` 不带 `-v` 是安全的 |
| 刷新 `/notes/123` 出现 404 | nginx 少了 SPA 回退 | `try_files $uri $uri/ /index.html;` |
| AI 讲解报 504 | LLM 生成超过网关默认超时 | nginx 已设 `proxy_read_timeout 300s` |
| PostgreSQL 报 `0xC0000142` | postmaster 进程上下文损坏 | `Get-Process postgres \| Stop-Process -Force` 后重启 |
| 找不到 go / pg_ctl | 装在非标准位置且没加 PATH | 复制 `local.config.example.ps1` 为 `local.config.ps1` 指定路径 |
| 端口被占用 | 80 / 8080 / 5173 被别的程序占了 | Docker 改 `WEB_PORT`；原生改 `.env` 里的 `HTTP_PORT` |

---

## 6. 关于密钥

`.env` 文件**不会**跟着 git 走，这是对的 —— 但意味着换机器要重新生成。

| 密钥 | 怎么来 |
|---|---|
| `JWT_SECRET` | `setup.ps1` 随机生成。**换了它，旧机器签发的 token 全部失效**（所有人需要重新登录），这是符合预期的 |
| `AI_INTERNAL_TOKEN` | `setup.ps1` 随机生成。两个服务必须一致，改了要同时改 |
| `CHAT_API_KEY` | **必须手工填**。这是 OpenCode Go 的 key，机器相关，在新机器上重新取一次 |
| GitHub OAuth 的 ID/Secret | 如果要用 GitHub 登录，去 [GitHub Settings → Developer settings](https://github.com/settings/developers) 新建一个 OAuth App，回调地址填新机器的 |

> **搬运 `.env` 的取舍**：直接拷旧机器的 `.env` 能省事，但
> ① 密钥跟着传播（泄露面变大）② 里面的 `localhost` 之类可能不适用。
> 建议重新生成，只手工补 `CHAT_API_KEY` 那一个。

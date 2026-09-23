# leetnote-api

LeetNote 的主服务 —— 用 Go + Gin 实现的高并发业务 API。

> 完整设计见 [`../docs/DESIGN.md`](../docs/DESIGN.md)，环境说明见 [`../docs/ENVIRONMENT.md`](../docs/ENVIRONMENT.md)。

## 快速开始

```powershell
# 1. 确保 PostgreSQL 已启动
D:\dev\pg-start.ps1

# 2. 准备配置
Copy-Item .env.example .env

# 3. 下载依赖
go mod tidy

# 4. 启动
.\dev.ps1 run          # 或： go run ./cmd/server

# 5. 验证
curl http://localhost:8080/health
```

预期输出：

```json
{
  "status": "ok",
  "service": "leetnote-api",
  "version": "dev",
  "commit": "none",
  "uptime": "5s",
  "database": "ok"
}
```

## 常用命令

本机没装 `make`，用 `dev.ps1` 代替（CI 里用 `Makefile`）：

| 命令 | 作用 |
|---|---|
| `.\dev.ps1 run` | 本地启动 |
| `.\dev.ps1 build` | 编译到 `bin/leetnote-api.exe`（注入版本号） |
| `.\dev.ps1 test` | 跑测试（含竞态检测 + 覆盖率） |
| `.\dev.ps1 cover` | 生成 HTML 覆盖率报告 |
| `.\dev.ps1 fmt` | 格式化 + 整理依赖 |
| `.\dev.ps1 lint` | golangci-lint 静态检查 |
| `.\dev.ps1 migrate-up` | 应用数据库迁移 |

## 目录结构

```
leetnote-api/
├── cmd/server/main.go          入口：装配依赖 + 优雅关闭
├── internal/
│   ├── config/                 Viper 风格的 env 配置加载
│   ├── logging/                slog 结构化日志
│   ├── db/                     pgx 连接池
│   ├── router/                 路由 + 中间件装配
│   ├── handler/                HTTP 处理层
│   ├── middleware/             RequestID / Logger / Recovery / CORS
│   ├── pkg/
│   │   ├── errs/               业务错误类型
│   │   └── response/           统一响应封装
│   └── version/                编译期注入的版本信息
├── migrations/                 golang-migrate SQL 文件
├── .env.example
├── Makefile                    CI / Linux 用
└── dev.ps1                     Windows 本地用
```

## 接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/health` | 完整健康检查（含数据库 ping） |
| GET | `/ready` | 就绪探针 |
| GET | `/live` | 存活探针 |

## Git 与远程仓库

模块路径已设为 `github.com/mzzzmnq/leetnote-api`，与 GitHub 用户名一致，无需再改。

首次推送到远程：

```powershell
cd D:\vibecoding_test\leetnote
git init
git add .
git commit -m "chore: 项目初始化（M1 Go 骨架 + 数据库迁移 + 设计文档）"

# 先在 GitHub 网页建一个空仓库 leetnote（不要勾选 README / .gitignore / license）
git branch -M main
git remote add origin https://github.com/mzzzmnq/leetnote.git
git push -u origin main
```

第一次 `push` 时 **Git Credential Manager 会弹出浏览器让你登录 GitHub**，
授权一次后凭据会被加密保存到 Windows 凭据管理器，后续推送不用再登录。

> 如果弹窗没出现或被拒绝，可以手动触发：`git credential-manager github login`

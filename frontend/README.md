# leetnote-frontend

LeetNote 的前端 —— Vue 3 + TypeScript + Vite。

> 完整设计见 [`../docs/DESIGN.md`](../docs/DESIGN.md)，环境说明见 [`../docs/ENVIRONMENT.md`](../docs/ENVIRONMENT.md)。

## 快速开始

```powershell
# 1. 起数据库和后端
D:\dev\pg-start.ps1
cd ..\leetnote-api && .\dev.ps1 run     # :8080

# 2. 起前端
cd ..\frontend
Copy-Item .env.example .env
npm install
npm run dev                             # :5173
```

打开 http://localhost:5173 即可。

## 常用命令

| 命令 | 作用 |
|---|---|
| `npm run dev` | 开发服务器（HMR） |
| `npm run build` | 类型检查 + 生产构建到 `dist/` |
| `npm run preview` | 预览构建产物 |
| `npm run typecheck` | 只做类型检查 |

## 目录结构

```
src/
├── main.ts                 入口：装 Pinia / Router，注入 401 处理器
├── App.vue                 Naive UI 的 Provider 包裹
├── api/
│   ├── client.ts           axios 实例 + 拦截器（注入 Token / 401 自动刷新）
│   ├── token.ts            access_token 的内存存储
│   ├── types.ts            与后端 DTO 一一对应的类型
│   ├── auth.ts             注册 / 登录 / 登出 / GitHub 授权
│   └── users.ts            个人资料 / 改密码 / 第三方账号
├── router/index.ts         路由表 + 登录守卫
├── stores/user.ts          Pinia：用户状态与会话恢复
├── layouts/DefaultLayout.vue
├── views/                  Login / Register / OAuthCallback / Dashboard / Notes / Settings / NotFound
└── styles/main.css
```

## 认证机制（前端侧）

| 数据 | 存放位置 | 为什么 |
|---|---|---|
| `access_token` | **内存**（`api/token.ts`） | 不写 localStorage，XSS 读不到 |
| `refresh_token` | **httpOnly Cookie**（后端下发） | 前端 JS 完全接触不到 |

**页面刷新后为什么还是登录态？**

`access_token` 在内存里，刷新就没了。但 `refresh_token` 在 Cookie 里还在，
所以路由守卫第一次执行时会调一次 `POST /auth/refresh` 把会话恢复回来。

**401 自动刷新（关键实现）**

`api/client.ts` 里做了两件事：

1. **单飞（single-flight）**：并发请求同时 401 时，只发起一次刷新，
   其余等待同一个 Promise。因为 refresh token 是**轮换**的，
   并发刷新会让后发的请求拿到已失效的 token，反而把自己踢下线。
2. **独立 axios 实例**：刷新请求走不带拦截器的 `bareClient`。
   否则刷新接口自己 401 时会再次触发刷新 → 无限递归。

## 关于 CORS

开发环境**不用 Vite 代理**，浏览器直连 `localhost:8080`。
目的是真实验证后端的 CORS + Cookie 配置——用代理会掩盖跨域问题。

> `localhost:5173` 与 `localhost:8080` 端口不同（跨域），但**同站**（SameSite 看的是站点不是端口），
> 所以 `SameSite=Lax` 的 Cookie 能正常携带。

## 已知问题

### TypeScript 必须锁在 5.x

`npm install typescript` 默认装 **7.x**（Go 重写版），但它改了包导出结构，
而 `vue-tsc@3` 内部依赖 `typescript/lib/tsc` 这个路径，会直接报错：

```
Error [ERR_PACKAGE_PATH_NOT_EXPORTED]: Package subpath './lib/tsc' is not defined
```

更坑的是 `vue-tsc` 的 `peerDependencies` 写着 `typescript: ">=5.0.0"`，
**声明上兼容但实际不兼容**。

所以 `package.json` 里锁了 `"typescript": "^5.9.3"`。等 `vue-tsc` 支持 TS 7 再升。

## 下一步

- **M3**：接上笔记 / 题目 / 解法 CRUD，替换 `NotesView` 的占位
- **M5**：Markdown 编辑器（Vditor）+ 代码高亮 + ECharts 统计看板

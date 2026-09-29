# =============================================================
# frontend（Vue 3 + Vite）
# =============================================================
#
# 多阶段：
#   构建阶段用 node 编译出静态文件
#   运行阶段只留 nginx + dist —— Node 本身**不进最终镜像**
#
# 这点很容易被忽略：一个 node:22 镜像 900MB+，而 nginx:alpine 只有 50MB。
# 运行时根本不需要 Node，把它留在镜像里纯属浪费。
# =============================================================

# ---------------------------------------------------------------
# 阶段一：构建
# ---------------------------------------------------------------
FROM node:22-alpine AS builder

WORKDIR /app

# 先拷依赖清单（含 lock 文件）
COPY package.json package-lock.json ./

# npm ci 而不是 npm install：
#   * 严格按 lock 文件装，保证每次构建的依赖树完全一致
#   * lock 和 package.json 不一致时直接报错，而不是悄悄改依赖
RUN npm ci

COPY . .

# 【编译期变量】Vite 会把 import.meta.env.* 在构建时**内联成字面量**，
# 所以 API 地址必须在 build 时就确定，运行时改环境变量没用。
#
# 这里默认给相对路径 /api/v1 —— 配合 nginx 同源代理，
# 前端不需要知道后端在哪，也就没有 CORS 和跨域 cookie 的问题。
ARG VITE_API_BASE_URL=/api/v1
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

RUN npm run build

# ---------------------------------------------------------------
# 阶段二：运行
# ---------------------------------------------------------------
FROM nginx:1.27-alpine

# 换掉默认站点配置（含 API 反代与 SPA history 回退）
COPY nginx.conf /etc/nginx/conf.d/default.conf

COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 \
    CMD wget -qO- http://127.0.0.1/ >/dev/null || exit 1

# 用官方镜像自带的 entrypoint，它会在启动时做配置模板处理
CMD ["nginx", "-g", "daemon off;"]

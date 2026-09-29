# =============================================================
# leetnote-api（Go 主服务）
# =============================================================
#
# 多阶段构建：
#   builder 阶段有完整的 Go SDK（约 800MB）
#   运行阶段只留一个静态二进制（最终镜像约 20MB）
#
# 这个镜像里放了【两个】二进制：
#   /server   主服务
#   /migrate  数据库迁移
# 两者用同一个镜像、不同 entrypoint —— compose 里 migrate 是一次性任务，
# 不需要为它单独构建一个镜像。
# =============================================================

# ---------------------------------------------------------------
# 阶段一：构建
# ---------------------------------------------------------------
FROM golang:1.27-alpine AS builder

WORKDIR /src

# 【顺序很关键】先只拷依赖清单。
# 这样只要 go.mod / go.sum 没变，下面这层 `go mod download` 就一直命中缓存。
# 如果把 COPY . . 放在前面，改一行业务代码都会触发重新下载全部依赖 ——
# 构建从 10 秒变成几分钟。
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 产出静态链接的二进制，
# 这样运行阶段可以不依赖 libc，换个底座也能跑。
# -s -w 去掉符号表和调试信息，二进制能小 30% 左右。
RUN CGO_ENABLED=0 GOOS=linux go build \
        -ldflags "-s -w" \
        -o /out/server ./cmd/server \
    && CGO_ENABLED=0 GOOS=linux go build \
        -ldflags "-s -w" \
        -o /out/migrate ./cmd/migrate

# ---------------------------------------------------------------
# 阶段二：运行
# ---------------------------------------------------------------
# 用 alpine 而不是 distroless：体积相差不大（5MB vs 2MB），
# 但 alpine 有 shell —— 出问题时 `docker compose exec api sh` 能进去看一眼。
# 容器里没有 shell 时排查问题会非常痛苦。
FROM alpine:3.21

# 时区与根证书：Go 服务要访问 LeetCode 拉题目数据（import 时）
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/server /app/server
COPY --from=builder /out/migrate /app/migrate
COPY --from=builder /src/migrations /app/migrations

# 非 root 运行。容器逃逸时攻击面小很多，也是绝大多数安全基线的硬性要求。
USER appuser

EXPOSE 8080

# 健康检查：编排工具靠它判断容器是否真的可用（不只是"进程还在"）
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 \
    CMD wget -qO- http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/app/server"]

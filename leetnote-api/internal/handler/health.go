package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mzzzmnq/leetnote-api/internal/version"
)

// HealthHandler 提供健康检查。
//
// 健康检查分两种，面试常被追问区别：
//   - liveness（存活探针）：进程还在吗？不查依赖，避免依赖抖动导致进程被反复重启
//   - readiness（就绪探针）：能对外服务吗？必须查数据库等关键依赖
//
// 这里的 /health 是 readiness 语义：数据库挂了就返回 503。
type HealthHandler struct {
	pool      *pgxpool.Pool
	startedAt time.Time
}

func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{
		pool:      pool,
		startedAt: time.Now(),
	}
}

type healthResponse struct {
	Status   string `json:"status"`
	Service  string `json:"service"`
	Version  string `json:"version"`
	Commit   string `json:"commit"`
	Uptime   string `json:"uptime"`
	Database string `json:"database"`
}

// Check GET /health
func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	status := "ok"
	code := http.StatusOK
	database := "ok"

	if err := h.pool.Ping(ctx); err != nil {
		status = "degraded"
		code = http.StatusServiceUnavailable
		database = "down"
	}

	c.JSON(code, healthResponse{
		Status:   status,
		Service:  "leetnote-api",
		Version:  version.Version,
		Commit:   version.Commit,
		Uptime:   time.Since(h.startedAt).Round(time.Second).String(),
		Database: database,
	})
}

// Ready GET /ready —— 纯就绪探针，依赖不健康时返回 503。
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.pool.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// Live GET /live —— 纯存活探针，永远返回 200（只要进程还能响应）。
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

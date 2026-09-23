package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mzzzmnq/leetnote-api/internal/config"
	"github.com/mzzzmnq/leetnote-api/internal/db"
	"github.com/mzzzmnq/leetnote-api/internal/logging"
	"github.com/mzzzmnq/leetnote-api/internal/router"
	"github.com/mzzzmnq/leetnote-api/internal/version"
)

func main() {
	if err := run(); err != nil {
		slog.Error("服务异常退出", "error", err)
		os.Exit(1)
	}
}

// run 返回 error 而不是直接 os.Exit，这样 defer 能正常执行（资源能释放）。
func run() error {
	// ---------- 1. 配置 ----------
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(cfg.AppEnv)
	slog.SetDefault(logger)

	slog.Info("正在启动 leetnote-api",
		"env", cfg.AppEnv,
		"version", version.Version,
		"commit", version.Commit,
	)

	// ---------- 2. 数据库 ----------
	pool, err := db.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	slog.Info("数据库连接池就绪", "max_conns", pool.Config().MaxConns)

	// ---------- 3. HTTP 服务 ----------
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: router.New(cfg, pool),
		// 这几个超时是防「慢连接攻击」的第一道防线，生产环境必须设
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("HTTP 服务已启动", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// ---------- 4. 优雅关闭 ----------
	// 收到 SIGINT(Ctrl+C) / SIGTERM 后：停止接收新请求 → 等待进行中的请求完成 → 退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("HTTP 服务启动失败: %w", err)
	case sig := <-quit:
		slog.Info("收到退出信号，开始优雅关闭", "signal", sig.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("优雅关闭超时: %w", err)
	}

	slog.Info("服务已安全退出")
	return nil
}

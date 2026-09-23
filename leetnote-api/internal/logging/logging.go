package logging

import (
	"log/slog"
	"os"
)

// New 根据运行环境构造 slog.Logger。
//
//   - 生产：JSON 格式，便于日志采集系统（ELK / Loki）解析
//   - 开发：Text 格式，人眼可读
//
// 用标准库的 log/slog 而不是第三方日志库：Go 1.21+ 自带，零依赖，
// 结构化日志能力完全够用。
func New(appEnv string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	var handler slog.Handler
	if appEnv == "production" {
		opts.Level = slog.LevelInfo
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

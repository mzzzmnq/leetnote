package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config 是应用的完整配置，全部来自环境变量（12-Factor 原则）。
type Config struct {
	AppEnv   string `envconfig:"APP_ENV" default:"development"`
	HTTPPort int    `envconfig:"HTTP_PORT" default:"8080"`
	GRPCPort int    `envconfig:"GRPC_PORT" default:"9091"`

	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`
	RedisAddr   string `envconfig:"REDIS_ADDR" default:"localhost:6379"`

	JWTSecret  string        `envconfig:"JWT_SECRET" required:"true"`
	AccessTTL  time.Duration `envconfig:"ACCESS_TOKEN_TTL" default:"15m"`
	RefreshTTL time.Duration `envconfig:"REFRESH_TOKEN_TTL" default:"168h"`

	AIGRPCAddr  string   `envconfig:"AI_GRPC_ADDR" default:"localhost:9090"`
	CORSOrigins []string `envconfig:"CORS_ORIGINS" default:"http://localhost:5173"`

	// FrontendURL 用于 OAuth 回调后跳回前端
	FrontendURL string `envconfig:"FRONTEND_URL" default:"http://localhost:5173"`

	// GitHub OAuth（留空则关闭该登录方式）
	GitHubClientID     string `envconfig:"GITHUB_CLIENT_ID"`
	GitHubClientSecret string `envconfig:"GITHUB_CLIENT_SECRET"`
	GitHubRedirectURL  string `envconfig:"GITHUB_REDIRECT_URL" default:"http://localhost:8080/api/v1/auth/github/callback"`
}

// IsProduction 用于判断是否开启 JSON 日志、Gin Release 模式等。
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// Load 读取 .env（可选）并解析环境变量到 Config。
//
// 注意：.env 不存在不算错误——生产环境直接注入真实环境变量即可。
func Load() (*Config, error) {
	_ = godotenv.Load(".env", "../.env")

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败（请检查 .env 或环境变量）: %w", err)
	}
	return &cfg, nil
}

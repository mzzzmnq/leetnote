package config

import (
	"fmt"
	"strings"
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

	// CookieSecure 控制 refresh cookie 是否带 Secure 属性（只有 HTTPS 才传）。
	//
	// 【为什么不直接用 IsProduction()】这两件事不是一回事：
	// "是不是生产环境" 决定日志格式、Gin 模式；
	// "cookie 要不要 Secure" 只取决于**当前是不是走 HTTPS**。
	//
	// 混在一起的后果很隐蔽：在 Docker 里用 http://localhost 访问、
	// 同时 APP_ENV=production 时，浏览器会因为 Secure 属性拒绝存储 cookie，
	// 表现为「登录接口返回 200，但刷新页面就掉登录」—— 很难查。
	//
	// 不填时的默认值见 IsCookieSecure()：生产环境为 true，其余为 false。
	CookieSecure string `envconfig:"COOKIE_SECURE"`

	AIGRPCAddr  string   `envconfig:"AI_GRPC_ADDR" default:"localhost:9090"`
	CORSOrigins []string `envconfig:"CORS_ORIGINS" default:"http://localhost:5173"`

	// ---------- leetnote-ai（Python 服务）----------
	// 目前用 HTTP/JSON 通信。原计划的 gRPC 留待后续（见 docs/DESIGN.md 说明）：
	// 这个数据量下 HTTP 完全够用，而引入 gRPC 需要 protoc 工具链与两侧代码生成，
	// 收益不足以抵消复杂度。
	AIServiceURL string `envconfig:"AI_SERVICE_URL" default:"http://127.0.0.1:8000"`
	// 服务间共享密钥，AI 服务用它拒绝非本服务发来的请求
	AIInternalToken string `envconfig:"AI_INTERNAL_TOKEN" default:"dev-internal-token-change-me"`

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

// IsCookieSecure 决定 refresh cookie 要不要加 Secure 属性。
//
// 显式配置优先；没配就跟随 APP_ENV（生产为 true）。
// 想在 production 下用纯 HTTP 访问时把它设成 "false"。
func (c *Config) IsCookieSecure() bool {
	switch strings.ToLower(strings.TrimSpace(c.CookieSecure)) {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	default:
		return c.IsProduction()
	}
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

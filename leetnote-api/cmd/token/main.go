// Command token 管理长期访问令牌（PAT）。
//
// 供本机工具 / 浏览器脚本（如 openleet）用它访问 API，无需交互式登录。
//
// 用法：
//
//	go run ./cmd/token -user mzzzmnq -name "openleet 插件"   # 新建并打印明文（只显示一次）
//	go run ./cmd/token -user mzzzmnq -list                  # 列出该用户的令牌
//	go run ./cmd/token -user mzzzmnq -revoke 3              # 吊销指定 id
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mzzzmnq/leetnote-api/internal/config"
	"github.com/mzzzmnq/leetnote-api/internal/model"
	"github.com/mzzzmnq/leetnote-api/internal/pkg/apitoken"
	"github.com/mzzzmnq/leetnote-api/internal/repository"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		username string
		name     string
		list     bool
		revokeID int64
	)
	flag.StringVar(&username, "user", "", "用户名（必填）")
	flag.StringVar(&name, "name", "", "令牌用途标签（新建时用）")
	flag.BoolVar(&list, "list", false, "列出该用户的令牌")
	flag.Int64Var(&revokeID, "revoke", 0, "吊销指定令牌 id")
	flag.Parse()

	if username == "" {
		flag.Usage()
		return fmt.Errorf("-user 是必填参数")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	defer pool.Close()

	users := repository.NewUserRepository(pool)
	tokens := repository.NewAPITokenRepository(pool)

	user, err := users.GetByLogin(ctx, username)
	if err != nil {
		return fmt.Errorf("找不到用户 %q: %w", username, err)
	}

	switch {
	case list:
		return listTokens(ctx, tokens, user)
	case revokeID > 0:
		if err := tokens.Revoke(ctx, user.ID, revokeID); err != nil {
			return err
		}
		fmt.Printf("✅ 已吊销令牌 #%d\n", revokeID)
		return nil
	default:
		return createToken(ctx, tokens, user, name)
	}
}

func createToken(ctx context.Context, tokens repository.APITokenRepository, user *model.User, name string) error {
	label := name
	if label == "" {
		label = "未命名"
	}

	raw, hash, err := apitoken.Generate()
	if err != nil {
		return err
	}
	if err := tokens.Create(ctx, &model.APIToken{
		UserID:    user.ID,
		Name:      label,
		TokenHash: hash,
	}); err != nil {
		return err
	}

	fmt.Printf("\n✅ 已为 %s 创建令牌「%s」\n", user.Username, label)
	fmt.Println("\n  令牌（只显示这一次，请立即保存）：")
	fmt.Printf("\n    %s\n\n", raw)
	fmt.Println("  用法：填进 openleet 的配置，或直接作为请求头：")
	fmt.Printf("    Authorization: Bearer %s\n\n", raw)
	return nil
}

func listTokens(ctx context.Context, tokens repository.APITokenRepository, user *model.User) error {
	list, err := tokens.ListByUser(ctx, user.ID)
	if err != nil {
		return err
	}
	if len(list) == 0 {
		fmt.Printf("用户 %s 还没有任何令牌。\n", user.Username)
		return nil
	}

	fmt.Printf("用户 %s 的令牌：\n", user.Username)
	for _, t := range list {
		last := "从未使用"
		if t.LastUsedAt != nil {
			last = t.LastUsedAt.Format("2006-01-02 15:04")
		}
		fmt.Printf("  #%-4d %-20s 创建 %s   最近使用 %s\n",
			t.ID, t.Name, t.CreatedAt.Format("2006-01-02"), last)
	}
	return nil
}

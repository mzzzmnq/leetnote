// 数据库迁移工具。
//
// 用法：
//
//	go run ./cmd/migrate up            # 应用所有未执行的迁移
//	go run ./cmd/migrate down [n]      # 回滚 n 步（默认 1 步）
//	go run ./cmd/migrate version       # 查看当前版本
//	go run ./cmd/migrate force <v>     # 强制把版本设为 v（基线化 / 修复脏状态）
//
// 【为什么把它做成项目里的子命令，而不是用 golang-migrate 的 CLI】
//
// 官方的 `migrate` CLI 需要单独安装（go install + 编译一大堆依赖），
// 换台机器就要重来一遍，很多人第一次 clone 下来会卡在这一步。
// 把它作为**库**引进来，`go run` 就能跑，只要有 Go 环境就行 ——
// 迁移逻辑仍然是最标准的那个实现，没有自己造轮子。
//
// 迁移文件是 migrations/ 下的版本化 SQL，命名规则 {版本}_{名称}.{up|down}.sql。
// 版本号与是否执行过记录在数据库的 schema_migrations 表里。
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	// 数据库驱动：用现有的 DATABASE_URL 即可，不需要额外的配置
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	// 迁移来源：本地文件系统中的 SQL 文件
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/mzzzmnq/leetnote-api/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var migrationsDir string
	flag.StringVar(&migrationsDir, "path", "migrations", "迁移文件所在目录")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		return usage()
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// file:// 源要求绝对路径（相对路径在不同工作目录下会找不到文件）
	abs, err := filepath.Abs(migrationsDir)
	if err != nil {
		return fmt.Errorf("解析迁移目录失败: %w", err)
	}
	if _, err := os.Stat(abs); err != nil {
		return fmt.Errorf("迁移目录不存在: %s", abs)
	}

	m, err := migrate.New("file://"+filepath.ToSlash(abs), cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("初始化迁移器失败: %w", err)
	}
	// m.Close 会返回 source 与 database 两个错误，只关心有没有出错
	defer func() { _, _ = m.Close() }()

	switch args[0] {
	case "up":
		return runUp(m)
	case "down":
		return runDown(m, args[1:])
	case "version":
		return runVersion(m)
	case "force":
		return runForce(m, args[1:])
	default:
		return usage()
	}
}

func runUp(m *migrate.Migrate) error {
	// 已经是最新时 migrate 会返回 ErrNoChange，
	// 这不是错误 —— 重复执行 up 本来就应该是安全的。
	err := m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		v, _, _ := m.Version()
		fmt.Printf("✅ 已是最新版本（%d），无需迁移\n", v)
		return nil
	}
	if err != nil {
		return fmt.Errorf("迁移失败: %w", err)
	}

	v, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("迁移已执行，但读取版本失败: %w", err)
	}
	fmt.Printf("✅ 迁移完成，当前版本 %d（dirty=%v）\n", v, dirty)
	return nil
}

func runDown(m *migrate.Migrate, args []string) error {
	steps := 1
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err != nil || n < 1 {
			return fmt.Errorf("回滚步数必须是正整数，收到 %q", args[0])
		}
		steps = n
	}

	// 先确认当前版本，否则「回滚成功」但实际什么都没回滚，会让人误判
	if v, _, err := m.Version(); err == nil {
		fmt.Printf("当前版本 %d，准备回滚 %d 步\n", v, steps)
	}

	err := m.Steps(-steps)
	if errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("✅ 已经是初始状态，无需回滚")
		return nil
	}
	if err != nil {
		return fmt.Errorf("回滚失败: %w", err)
	}

	v, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		fmt.Println("✅ 已回滚到初始状态（所有迁移都已撤销）")
		return nil
	}
	if err != nil {
		return fmt.Errorf("回滚已执行，但读取版本失败: %w", err)
	}
	fmt.Printf("✅ 回滚完成，当前版本 %d（dirty=%v）\n", v, dirty)
	return nil
}

func runVersion(m *migrate.Migrate) error {
	v, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		fmt.Println("当前没有任何已执行的迁移（空库）")
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取版本失败: %w", err)
	}

	fmt.Printf("当前版本: %d\n", v)
	if dirty {
		fmt.Println("⚠️  状态为 dirty —— 上一次迁移中途失败了。")
		fmt.Println("    手工修好数据库后，用 `force <版本>` 清除这个标记。")
	}
	return nil
}

func runForce(m *migrate.Migrate, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("force 需要一个版本号参数")
	}
	v, err := strconv.Atoi(args[0])
	if err != nil || v < 0 {
		return fmt.Errorf("版本号必须是非负整数，收到 %q", args[0])
	}

	// force 只改 schema_migrations 里的记录，【不会】真正执行或撤销任何 SQL。
	// 典型用途：数据库结构已经手工建好（或迁移中途失败已修好），
	// 把版本对齐到实际状态，避免下次 up 重复执行已生效的迁移。
	if err := m.Force(v); err != nil {
		return fmt.Errorf("设置版本失败: %w", err)
	}
	fmt.Printf("✅ 已把版本强制设为 %d（未执行任何 SQL）\n", v)
	return nil
}

func usage() error {
	return fmt.Errorf(`用法: go run ./cmd/migrate <命令> [参数]

命令:
  up             应用所有未执行的迁移
  down [n]       回滚 n 步（默认 1 步）
  version        查看当前版本
  force <v>      强制把版本设为 v（不执行任何 SQL，用于基线化或修复脏状态）

示例:
  go run ./cmd/migrate up
  go run ./cmd/migrate down 1
  go run ./cmd/migrate force 6`)
}

package version

// 这些变量在编译时通过 -ldflags 注入：
//
//	go build -ldflags "-X github.com/mzzzmnq/leetnote-api/internal/version.Version=v0.1.0" ./cmd/server
//
// 目的是让线上跑着的二进制能自报版本，排查问题时非常有用。
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

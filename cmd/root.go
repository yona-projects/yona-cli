// Package cmd는 yona CLI의 Cobra 명령 트리를 구성한다.
package cmd

import (
	"github.com/search5/yona-cli/internal/api"
	"github.com/search5/yona-cli/internal/config"
	"github.com/spf13/cobra"
)

// Version은 "yona --version"이 출력하는 값이다. 릴리즈 빌드에서는 goreleaser 등이
// `-ldflags "-X github.com/search5/yona-cli/cmd.Version=..."`로 주입할 수 있도록 변수로
// 남겨둔다(Step 11 배포 작업 범위, 이번 라운드는 하드코딩 기본값만 둔다).
var Version = "0.1.0-dev"

// cmdContext는 루트 커맨드의 전역 플래그(--server/--token)를 하위 명령들과 공유한다.
// 패키지 전역 변수 대신 클로저로 캡처된 지역 변수를 쓰는 이유는 NewRootCmd()를 호출할 때마다
// 완전히 독립된 플래그 상태를 갖게 해 테스트 간 상태 누수를 막기 위함이다.
type cmdContext struct {
	server *string
	token  *string
}

// newClient는 --server/--token 플래그, 환경변수(YONA_HOST/YONA_TOKEN), 설정 파일의 순서로
// 접속 정보를 확정해 api.Client를 만든다(internal/config.ResolveToken 참고).
func (c *cmdContext) newClient() (*api.Client, error) {
	server, token, err := config.ResolveToken(*c.server, *c.token)
	if err != nil {
		return nil, err
	}
	return api.NewClient(server, token), nil
}

// NewRootCmd는 yona CLI의 전체 명령 트리를 만든다. main.go와 테스트 코드 양쪽에서 쓴다.
func NewRootCmd() *cobra.Command {
	ctx := &cmdContext{server: new(string), token: new(string)}

	root := &cobra.Command{
		Use:           "yona",
		Short:         "yuna 서버(REST API)를 감싸는 커맨드라인 도구",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.PersistentFlags().StringVar(ctx.server, "server", "", "yona 서버 URL (기본값: 설정 파일의 현재 서버 또는 YONA_HOST 환경변수)")
	root.PersistentFlags().StringVar(ctx.token, "token", "", "이 호출에만 쓸 API 토큰 (기본값: 설정 파일에 저장된 토큰 또는 YONA_TOKEN 환경변수) — 범위가 제한된 Fine-grained 토큰을 쓰고 싶을 때 지정한다")

	root.AddCommand(newAuthCmd(ctx))
	root.AddCommand(newIssueCmd(ctx))
	root.AddCommand(newPRCmd(ctx))
	root.AddCommand(newProjectCmd(ctx))
	root.AddCommand(newLabelCmd(ctx))
	root.AddCommand(newSearchCmd(ctx))
	root.AddCommand(newOrgCmd(ctx))
	root.AddCommand(newServerCmd(ctx))
	root.AddCommand(newBrowseCmd(ctx))
	root.AddCommand(newAdminCmd(ctx))
	root.AddCommand(newAPICmd(ctx))
	root.AddCommand(newStatusCmd(ctx))

	// "completion" 서브커맨드는 Cobra가 서브커맨드를 가진 루트 커맨드에 자동으로 등록한다
	// (ExecuteC() -> InitDefaultCompletionCmd(), CompletionOptions.DisableDefaultCmd 기본값
	// false) — 별도 구현이 필요 없다.
	return root
}

// Execute는 main.go의 진입점이다.
func Execute() error {
	return NewRootCmd().Execute()
}

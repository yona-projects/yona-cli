package cmd

import (
	"fmt"

	"github.com/search5/yona-cli/internal/config"
	"github.com/spf13/cobra"
)

// newServerCmd는 "yona server list/use" — gh CLI의 "gh auth switch"에 대응하되, yuna는
// 자체호스팅이라 회사/개인마다 완전히 다른 인스턴스를 오갈 일이 많아 "auth"가 아닌 별도
// 최상위 커맨드로 뒀다(yona-wiki P3-02 계획 문서 결정). "use"는 이미 로그인된 호스트로
// current_host만 전환할 뿐 재로그인을 요구하지 않는다 — config.UseHost() 참고.
func newServerCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "여러 yona 서버 사이를 전환한다",
	}
	cmd.AddCommand(newServerListCmd(ctx))
	cmd.AddCommand(newServerUseCmd(ctx))
	return cmd
}

func newServerListCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "로그인된 서버 목록을 보여준다 (현재 서버는 * 표시)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if len(cfg.Hosts) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "로그인된 yona 서버가 없습니다. 'yona auth login'을 실행하세요.")
				return nil
			}
			// auth status와 동일한 마스킹 방식(maskToken)을 재사용한다.
			for server, host := range cfg.Hosts {
				marker := "  "
				if server == cfg.CurrentHost {
					marker = "* "
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", marker, server)
				fmt.Fprintf(cmd.OutOrStdout(), "    토큰: %s\n", maskToken(host.Token))
			}
			return nil
		},
	}
	return cmd
}

func newServerUseCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use <server>",
		Short: "이미 로그인된 서버로 전환한다 (재로그인 불필요)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := cfg.UseHost(args[0]); err != nil {
				return err
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "현재 서버를 %s(으)로 전환했습니다.\n", args[0])
			return nil
		},
	}
	return cmd
}

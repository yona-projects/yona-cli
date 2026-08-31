package cmd

import (
	"fmt"

	"github.com/search5/yona-cli/internal/config"
	"github.com/search5/yona-cli/internal/gitutil"
	"github.com/search5/yona-cli/internal/weburl"
	"github.com/spf13/cobra"
)

// newBrowseCmd는 "gh browse" 대응 — 서버 API 호출 없이 URL만 계산해 브라우저로 연다.
// 인자가 없으면 현재 프로젝트 홈, "issue <n>"/"pr <n>"이면 해당 상세 페이지를 연다.
func newBrowseCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "browse [issue <number> | pr <number>]",
		Short: "현재 프로젝트/이슈/풀 리퀘스트를 브라우저로 연다",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			server, err := config.ResolveServer(*ctx.server)
			if err != nil {
				return err
			}

			target := weburl.Project(server, owner, project)
			if len(args) > 0 {
				if len(args) != 2 {
					return fmt.Errorf(`사용법: yona browse [issue <번호> | pr <번호>]`)
				}
				number, err := parseNumberArg(args[1])
				if err != nil {
					return err
				}
				switch args[0] {
				case "issue":
					target = weburl.Issue(server, owner, project, number)
				case "pr":
					target = weburl.PullRequest(server, owner, project, number)
				default:
					return fmt.Errorf(`알 수 없는 대상입니다: %q ("issue" 또는 "pr"만 지원)`, args[0])
				}
			}

			if err := gitutil.OpenInBrowser(target); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), target)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	return cmd
}

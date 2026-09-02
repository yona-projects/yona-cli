package cmd

import (
	"fmt"
	"io"

	"github.com/search5/yona-cli/internal/api"
	"github.com/spf13/cobra"
)

// newStatusCmd는 "gh status" 대응 — 최상위 커맨드로 서브커맨드 없이 바로 실행된다(gh status와
// 동일한 UX). yona-wiki P3-02 CLI `gh` 명령 체계 대조 감사에서 뒤늦게 발견된 갭(최초 감사표엔
// 없었다)이다. 이미 있던 `issue status`(담당/작성 이슈만)의 상위 집합으로, 담당 PR/리뷰요청 PR/
// 멘션된 이슈/저장소 활동까지 한 화면에 보여준다. gh status 자체가 -e/--exclude, -o/--org 외에는
// --json이나 -L 같은 공용 플래그를 안 받으므로(`gh status --help` 실측 확인) 이 커맨드도 --json
// 필드선택만 지원하고 페이지네이션/상태 필터는 두지 않는다(서버가 항상 열림 상태만 내려줌).
func newStatusCmd(ctx *cmdContext) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "내가 담당/리뷰요청받은 이슈·PR과 구독 중인 저장소 활동을 한 화면에 보여준다",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			status, err := client.GetUserStatus(cmd.Context())
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, status, jsonFields)
			}

			out := cmd.OutOrStdout()
			printStatusSection(out, "담당 이슈 (Assigned Issues)", status.AssignedIssues)
			printStatusSection(out, "담당 풀 리퀘스트 (Assigned Pull Requests)", status.AssignedPullRequests)
			printStatusSection(out, "리뷰 요청 (Review Requests)", status.ReviewRequests)
			printStatusSection(out, "멘션된 이슈 (Mentions, 이슈만 지원)", status.MentionedIssues)

			fmt.Fprintf(out, "\n저장소 활동 (Repository Activity)\n")
			if len(status.RepositoryActivity) == 0 {
				fmt.Fprintf(out, "  (없음)\n")
			}
			for _, activity := range status.RepositoryActivity {
				fmt.Fprintf(out, "  [%s] %s\n", activity.EventType, activity.Title)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json assignedIssues,reviewRequests)")
	return cmd
}

func printStatusSection(out io.Writer, heading string, section api.StatusSection) {
	fmt.Fprintf(out, "%s (열림 %d / 닫힘 %d)\n", heading, section.OpenCount, section.ClosedCount)
	if len(section.Items) == 0 {
		fmt.Fprintf(out, "  (없음)\n")
	}
	for _, item := range section.Items {
		fmt.Fprintf(out, "  #%s\t%s\n", num(item, "number"), str(item, "title"))
	}
}

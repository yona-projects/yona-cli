package cmd

import (
	"fmt"

	"github.com/search5/yona-cli/internal/api"
	"github.com/spf13/cobra"
)

func newPRCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "풀 리퀘스트 조회/생성/머지/리뷰",
	}
	cmd.AddCommand(newPRListCmd(ctx))
	cmd.AddCommand(newPRViewCmd(ctx))
	cmd.AddCommand(newPRCreateCmd(ctx))
	cmd.AddCommand(newPRMergeCmd(ctx))
	cmd.AddCommand(newPRReviewCmd(ctx))
	return cmd
}

func newPRListCmd(ctx *cmdContext) *cobra.Command {
	var repo, state string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "풀 리퀘스트 목록을 조회한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := parseRepo(repo)
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			prs, err := client.ListPullRequests(cmd.Context(), owner, project, state)
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(cmd, prs)
			}
			if len(prs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "풀 리퀘스트가 없습니다.")
				return nil
			}
			for _, pr := range prs {
				fmt.Fprintf(cmd.OutOrStdout(), "#%s\t%s\t%s\n", num(pr, "number"), str(pr, "state"), str(pr, "title"))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (필수)`)
	cmd.Flags().StringVar(&state, "state", "", "상태로 필터링 (예: OPEN, CLOSED, MERGED)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "서버 응답을 가공하지 않고 JSON 그대로 출력")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func newPRViewCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "view <number>",
		Short: "풀 리퀘스트 하나를 조회한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := parseRepo(repo)
			if err != nil {
				return err
			}
			number, err := parseNumberArg(args[0])
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			pr, err := client.GetPullRequest(cmd.Context(), owner, project, number)
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(cmd, pr)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "#%s %s\n상태: %s\n%s -> %s\n\n%s\n",
				num(pr, "number"), str(pr, "title"), str(pr, "state"), str(pr, "fromBranch"), str(pr, "toBranch"), str(pr, "body"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (필수)`)
	cmd.Flags().BoolVar(&asJSON, "json", false, "서버 응답을 가공하지 않고 JSON 그대로 출력")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func newPRCreateCmd(ctx *cmdContext) *cobra.Command {
	var repo, title, body, fromBranch, toBranch string
	var fromProjectID int64
	cmd := &cobra.Command{
		Use:   "create",
		Short: "새 풀 리퀘스트를 만든다",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := parseRepo(repo)
			if err != nil {
				return err
			}
			if title == "" || fromBranch == "" || toBranch == "" || fromProjectID == 0 {
				return fmt.Errorf("--title, --from-project-id, --from-branch, --to-branch는 모두 필수입니다")
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			pr, err := client.CreatePullRequest(cmd.Context(), owner, project, api.CreatePullRequestRequest{
				Title: title, Body: body, FromProjectID: fromProjectID, FromBranch: fromBranch, ToBranch: toBranch,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "풀 리퀘스트 #%s 생성됨: %s\n", num(pr, "number"), str(pr, "title"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상(to) 프로젝트, "owner/project" 형식 (필수)`)
	cmd.Flags().StringVar(&title, "title", "", "제목 (필수)")
	cmd.Flags().StringVar(&body, "body", "", "본문")
	cmd.Flags().Int64Var(&fromProjectID, "from-project-id", 0, "fork(from) 프로젝트의 숫자 ID (필수, 'yona project view'로 id 확인)")
	cmd.Flags().StringVar(&fromBranch, "from-branch", "", "fork(from) 브랜치 (필수)")
	cmd.Flags().StringVar(&toBranch, "to-branch", "", "대상(to) 브랜치 (필수)")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("from-project-id")
	_ = cmd.MarkFlagRequired("from-branch")
	_ = cmd.MarkFlagRequired("to-branch")
	return cmd
}

func newPRMergeCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "merge <number>",
		Short: "풀 리퀘스트를 머지한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := parseRepo(repo)
			if err != nil {
				return err
			}
			number, err := parseNumberArg(args[0])
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			result, err := client.MergePullRequest(cmd.Context(), owner, project, number)
			if err != nil {
				return err
			}
			if conflicts, ok := result["conflicts"].(bool); ok && conflicts {
				fmt.Fprintf(cmd.OutOrStdout(), "풀 리퀘스트 #%d에 충돌이 있어 머지되지 않았습니다.\n", number)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "풀 리퀘스트 #%d을(를) 머지했습니다.\n", number)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (필수)`)
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

// yona-wiki 계획 문서 주의사항 그대로: 이 서버의 addReviewer는 "리뷰어를 지정"하는 게 아니라
// 인증된 본인을 리뷰어로 자기등록하는 동작이다(PullRequestController.addReviewer 참고).
func newPRReviewCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "review <number>",
		Short: "본인을 해당 풀 리퀘스트의 리뷰어로 등록한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := parseRepo(repo)
			if err != nil {
				return err
			}
			number, err := parseNumberArg(args[0])
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			if err := client.AddReviewer(cmd.Context(), owner, project, number); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "풀 리퀘스트 #%d의 리뷰어로 등록되었습니다.\n", number)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (필수)`)
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newSearchCmd는 "gh search issues/prs/repos" 대응 — yona-wiki P3-02 4라운드가 추가한
// /api/v1/search/issues, /api/v1/search/projects와 7라운드가 추가한 /api/v1/search/prs를
// 그대로 감싼다.
//
// "yona search prs"는 한동안 이 CLI에 의도적으로 구현하지 않았었다 — 착수 당시(4라운드) yona
// SearchType enum에 PR을 색인하는 값이 없어 서버 자체에 대응 기능이 없었기 때문. 7라운드가
// 서버에 SearchType.PULL_REQUEST + GET /api/v1/search/prs를 추가했는데 CLI 배선이 따라가지 못한
// 채 남아있던 갭을 12라운드에서 메웠다.
func newSearchCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "이슈/풀 리퀘스트/프로젝트 통합검색",
	}
	cmd.AddCommand(newSearchIssuesCmd(ctx))
	cmd.AddCommand(newSearchProjectsCmd(ctx))
	cmd.AddCommand(newSearchPullRequestsCmd(ctx))
	return cmd
}

func newSearchIssuesCmd(ctx *cmdContext) *cobra.Command {
	var page, size int
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "issues <query>",
		Short: "이슈를 검색한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			result, err := client.SearchIssues(cmd.Context(), args[0], page, size)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, result.Content, jsonFields)
			}
			if len(result.Content) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "검색 결과가 없습니다.")
				return nil
			}
			for _, issue := range result.Content {
				fmt.Fprintf(cmd.OutOrStdout(), "#%s\t%s\n", num(issue, "number"), str(issue, "title"))
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&page, "page", 0, "페이지 번호 (0부터 시작)")
	cmd.Flags().IntVar(&size, "size", 0, "페이지 크기 (생략 시 서버 기본값)")
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json number,title)")
	return cmd
}

func newSearchProjectsCmd(ctx *cmdContext) *cobra.Command {
	var page, size int
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "projects <query>",
		Short: "프로젝트를 검색한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			result, err := client.SearchProjects(cmd.Context(), args[0], page, size)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, result.Content, jsonFields)
			}
			if len(result.Content) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "검색 결과가 없습니다.")
				return nil
			}
			for _, project := range result.Content {
				fmt.Fprintf(cmd.OutOrStdout(), "%s/%s\n", str(project, "owner"), str(project, "name"))
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&page, "page", 0, "페이지 번호 (0부터 시작)")
	cmd.Flags().IntVar(&size, "size", 0, "페이지 크기 (생략 시 서버 기본값)")
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json owner,name)")
	return cmd
}

func newSearchPullRequestsCmd(ctx *cmdContext) *cobra.Command {
	var page, size int
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "prs <query>",
		Short: "풀 리퀘스트를 검색한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			result, err := client.SearchPullRequests(cmd.Context(), args[0], page, size)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, result.Content, jsonFields)
			}
			if len(result.Content) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "검색 결과가 없습니다.")
				return nil
			}
			for _, pr := range result.Content {
				fmt.Fprintf(cmd.OutOrStdout(), "#%s\t%s\n", num(pr, "number"), str(pr, "title"))
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&page, "page", 0, "페이지 번호 (0부터 시작)")
	cmd.Flags().IntVar(&size, "size", 0, "페이지 크기 (생략 시 서버 기본값)")
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json number,title)")
	return cmd
}

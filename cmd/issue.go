package cmd

import (
	"fmt"

	"github.com/search5/yona-cli/internal/api"
	"github.com/spf13/cobra"
)

func newIssueCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "이슈 조회/생성/댓글/클로즈",
	}
	cmd.AddCommand(newIssueListCmd(ctx))
	cmd.AddCommand(newIssueViewCmd(ctx))
	cmd.AddCommand(newIssueCreateCmd(ctx))
	cmd.AddCommand(newIssueCommentCmd(ctx))
	cmd.AddCommand(newIssueCloseCmd(ctx))
	return cmd
}

func newIssueListCmd(ctx *cmdContext) *cobra.Command {
	var repo, state string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "이슈 목록을 조회한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := parseRepo(repo)
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			page, err := client.ListIssues(cmd.Context(), owner, project, state)
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(cmd, page)
			}
			if len(page.Content) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "이슈가 없습니다.")
				return nil
			}
			for _, issue := range page.Content {
				fmt.Fprintf(cmd.OutOrStdout(), "#%s\t%s\t%s\n", num(issue, "number"), str(issue, "state"), str(issue, "title"))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (필수)`)
	cmd.Flags().StringVar(&state, "state", "", "상태로 필터링 (예: OPEN, CLOSED)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "서버 응답을 가공하지 않고 JSON 그대로 출력")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func newIssueViewCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "view <number>",
		Short: "이슈 하나를 조회한다",
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
			issue, err := client.GetIssue(cmd.Context(), owner, project, number)
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(cmd, issue)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "#%s %s\n상태: %s\n\n%s\n", num(issue, "number"), str(issue, "title"), str(issue, "state"), str(issue, "body"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (필수)`)
	cmd.Flags().BoolVar(&asJSON, "json", false, "서버 응답을 가공하지 않고 JSON 그대로 출력")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func newIssueCreateCmd(ctx *cmdContext) *cobra.Command {
	var repo, title, body string
	var draft bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "새 이슈를 만든다",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := parseRepo(repo)
			if err != nil {
				return err
			}
			if title == "" {
				return fmt.Errorf("--title은 필수입니다")
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			issue, err := client.CreateIssue(cmd.Context(), owner, project, api.CreateIssueRequest{
				Title: title, Body: body, IsDraft: draft,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "이슈 #%s 생성됨: %s\n", num(issue, "number"), str(issue, "title"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (필수)`)
	cmd.Flags().StringVar(&title, "title", "", "이슈 제목 (필수)")
	cmd.Flags().StringVar(&body, "body", "", "이슈 본문")
	cmd.Flags().BoolVar(&draft, "draft", false, "초안(DRAFT)으로 생성")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

func newIssueCommentCmd(ctx *cmdContext) *cobra.Command {
	var repo, body string
	cmd := &cobra.Command{
		Use:   "comment <number>",
		Short: "이슈에 댓글을 남긴다",
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
			if body == "" {
				return fmt.Errorf("--body는 필수입니다")
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			if _, err := client.AddIssueComment(cmd.Context(), owner, project, number, api.CommentRequest{Contents: body}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "이슈 #%d에 댓글을 남겼습니다.\n", number)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (필수)`)
	cmd.Flags().StringVar(&body, "body", "", "댓글 내용 (필수)")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("body")
	return cmd
}

func newIssueCloseCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "close <number>",
		Short: "이슈를 닫는다",
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
			if _, err := client.CloseIssue(cmd.Context(), owner, project, number); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "이슈 #%d을(를) 닫았습니다.\n", number)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (필수)`)
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

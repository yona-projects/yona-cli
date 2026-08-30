package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newProjectCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "프로젝트(저장소) 조회",
	}
	cmd.AddCommand(newProjectListCmd(ctx))
	cmd.AddCommand(newProjectViewCmd(ctx))
	return cmd
}

func newProjectListCmd(ctx *cmdContext) *cobra.Command {
	var owner string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list <owner>",
		Short: "owner 아래의 프로젝트 목록을 조회한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner = args[0]
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			projects, err := client.ListProjects(cmd.Context(), owner)
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(cmd, projects)
			}
			if len(projects) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "프로젝트가 없습니다.")
				return nil
			}
			for _, p := range projects {
				fmt.Fprintf(cmd.OutOrStdout(), "%s/%s\t%s\n", p.Owner, p.Name, p.Scope)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "서버 응답을 가공하지 않고 JSON 그대로 출력")
	return cmd
}

// yona-wiki P3-02 계획 문서의 명령 예시("yona project view <name>")는 프로젝트 이름 하나만
// 받는 것처럼 보이지만, 서버 API(GET /api/v1/projects/{owner}/{project})는 owner도 필요하다.
// gh CLI의 "-R owner/repo" 관례를 그대로 따라 <owner/project> 형식의 단일 인자로 받는다.
func newProjectViewCmd(ctx *cmdContext) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "view <owner/project>",
		Short: "프로젝트 하나를 조회한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := parseRepo(args[0])
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			p, err := client.GetProject(cmd.Context(), owner, project)
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(cmd, p)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s/%s (id=%d)\nVCS: %s\n범위: %s\n\n%s\n", p.Owner, p.Name, p.ID, p.VCS, p.Scope, p.Overview)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "서버 응답을 가공하지 않고 JSON 그대로 출력")
	return cmd
}

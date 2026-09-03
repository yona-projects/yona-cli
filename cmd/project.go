package cmd

import (
	"fmt"

	"github.com/yona-projects/yona-cli/internal/api"
	"github.com/yona-projects/yona-cli/internal/config"
	"github.com/yona-projects/yona-cli/internal/gitutil"
	"github.com/yona-projects/yona-cli/internal/weburl"
	"github.com/spf13/cobra"
)

func newProjectCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "프로젝트(저장소) 조회/생성/수정/삭제/fork",
	}
	cmd.AddCommand(newProjectListCmd(ctx))
	cmd.AddCommand(newProjectViewCmd(ctx))
	cmd.AddCommand(newProjectCreateCmd(ctx))
	cmd.AddCommand(newProjectForkCmd(ctx))
	cmd.AddCommand(newProjectEditCmd(ctx))
	cmd.AddCommand(newProjectDeleteCmd(ctx))
	return cmd
}

func newProjectListCmd(ctx *cmdContext) *cobra.Command {
	var jsonFields string
	var limit int
	cmd := &cobra.Command{
		Use:   "list <owner>",
		Short: "owner 아래의 프로젝트 목록을 조회한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner := args[0]
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			projects, err := client.ListProjects(cmd.Context(), owner)
			if err != nil {
				return err
			}
			// 서버 GET /api/v1/projects/{owner}는 페이지네이션이 없어(List<Project> 전체 반환)
			// -L/--limit은 클라이언트 사이드 슬라이싱으로 처리한다.
			if limit > 0 && limit < len(projects) {
				projects = projects[:limit]
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, projects, jsonFields)
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
	cmd.Flags().IntVarP(&limit, "limit", "L", 0, "가져올 최대 개수 (클라이언트 사이드 슬라이싱, 생략 시 전체)")
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json owner,name,scope)")
	return cmd
}

// yona-wiki P3-02 계획 문서의 명령 예시("yona project view <name>")는 프로젝트 이름 하나만
// 받는 것처럼 보이지만, 서버 API(GET /api/v1/projects/{owner}/{project})는 owner도 필요하다.
// gh CLI의 "-R owner/repo" 관례를 그대로 따라 <owner/project> 형식의 단일 인자로 받는다.
func newProjectViewCmd(ctx *cmdContext) *cobra.Command {
	var jsonFields string
	var web bool
	cmd := &cobra.Command{
		Use:   "view <owner/project>",
		Short: "프로젝트 하나를 조회한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := parseRepo(args[0])
			if err != nil {
				return err
			}
			if web {
				server, err := config.ResolveServer(*ctx.server)
				if err != nil {
					return err
				}
				return gitutil.OpenInBrowser(weburl.Project(server, owner, project))
			}

			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			p, err := client.GetProject(cmd.Context(), owner, project)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, p, jsonFields)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s/%s (id=%d)\nVCS: %s\n범위: %s\n\n%s\n", p.Owner, p.Name, p.ID, p.VCS, p.Scope, p.Overview)
			return nil
		},
	}
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json owner,name,scope)")
	cmd.Flags().BoolVar(&web, "web", false, "API 호출 대신 프로젝트 홈 웹 페이지를 브라우저로 연다")
	return cmd
}

// newProjectCreateCmd는 "gh repo create" 대응 — yona-wiki P3-02 4라운드가 추가한
// POST /api/v1/projects(bare)를 그대로 감싼다. 이 경로는 어떤 Fine-grained 스코프 패턴과도
// 매칭되지 않아 세션 로그인/레거시 전권 토큰으로만 성공한다(계획 문서에 의도적 설계로 기록됨).
func newProjectCreateCmd(ctx *cmdContext) *cobra.Command {
	var overview, scope, vcs string
	cmd := &cobra.Command{
		Use:   "create <owner/name>",
		Short: "새 프로젝트를 만든다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := parseRepo(args[0])
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			p, err := client.CreateProject(cmd.Context(), api.CreateProjectRequest{
				Owner: owner, Name: name, Overview: overview, ProjectScope: scope, VCS: vcs,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "프로젝트 %s/%s 생성됨 (id=%d)\n", p.Owner, p.Name, p.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&overview, "overview", "", "프로젝트 설명")
	cmd.Flags().StringVar(&scope, "scope", "PUBLIC", "공개 범위 (PUBLIC 또는 PRIVATE)")
	cmd.Flags().StringVar(&vcs, "vcs", "GIT", "버전관리시스템 (예: GIT)")
	return cmd
}

// newProjectForkCmd는 "gh repo fork" 대응 — yona-wiki P3-02 4라운드가 추가한
// POST /api/v1/projects/{owner}/{project}/fork를 그대로 감싼다.
func newProjectForkCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fork <owner/project>",
		Short: "프로젝트를 자신의 계정 아래로 fork한다",
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
			forked, err := client.ForkProject(cmd.Context(), owner, project)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s/%s을(를) %s/%s(으)로 fork했습니다.\n", owner, project, str(forked, "owner"), str(forked, "name"))
			return nil
		},
	}
	return cmd
}

// newProjectEditCmd는 "gh repo edit" 대응 — yona-wiki P3-02 4라운드가 추가한
// PATCH /api/v1/projects/{owner}/{project}/settings를 그대로 감싼다. overview/projectScope는
// 서버에서 non-null 필수값이라(UpdateProjectRequest), 플래그로 지정하지 않으면 read-modify-write로
// 현재 값을 GetProject()에서 읽어와 그대로 유지한다.
func newProjectEditCmd(ctx *cmdContext) *cobra.Command {
	var name, overview, scope, defaultBranch string
	cmd := &cobra.Command{
		Use:   "edit <owner/project>",
		Short: "프로젝트 설정을 수정한다",
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
			current, err := client.GetProject(cmd.Context(), owner, project)
			if err != nil {
				return err
			}
			req := api.UpdateProjectRequest{
				Overview:     current.Overview,
				ProjectScope: current.Scope,
			}
			if overview != "" {
				req.Overview = overview
			}
			if scope != "" {
				req.ProjectScope = scope
			}
			if name != "" {
				req.Name = &name
			}
			if defaultBranch != "" {
				req.DefaultBranch = &defaultBranch
			}
			if _, err := client.UpdateProject(cmd.Context(), owner, project, req); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s/%s 설정을 수정했습니다.\n", owner, project)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "새 프로젝트 이름(개명)")
	cmd.Flags().StringVar(&overview, "overview", "", "새 설명")
	cmd.Flags().StringVar(&scope, "scope", "", "새 공개 범위 (PUBLIC 또는 PRIVATE)")
	cmd.Flags().StringVar(&defaultBranch, "default-branch", "", "새 기본 브랜치")
	return cmd
}

// newProjectDeleteCmd는 "gh repo delete" 대응 — yona-wiki P3-02 4라운드가 추가한
// DELETE /api/v1/projects/{owner}/{project}/settings를 그대로 감싼다. gh repo delete와 동일하게
// 되돌릴 수 없는 작업이라 --yes로 명시적 확인을 요구한다.
func newProjectDeleteCmd(ctx *cmdContext) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <owner/project>",
		Short: "프로젝트를 삭제한다 (되돌릴 수 없음)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := parseRepo(args[0])
			if err != nil {
				return err
			}
			if !yes {
				return fmt.Errorf("되돌릴 수 없는 작업입니다 — 정말 삭제하려면 --yes를 지정하세요")
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			if err := client.DeleteProject(cmd.Context(), owner, project); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s/%s을(를) 삭제했습니다.\n", owner, project)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "확인 없이 삭제를 진행한다 (필수)")
	return cmd
}

package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/search5/yona-cli/internal/api"
	"github.com/spf13/cobra"
)

// yona-wiki P3-02 Step9 조사 결과(internal/api/admin.go 상단 주석 참고): 백업은 실제 JSON에
// 가까운 API가 있어 완전히 연결했지만, 웹훅/권한 관리는 서버에 세션/폼 기반 레거시 컨트롤러만
// 있고 owner/project 기반 3종(생성/변경/삭제)은 연결 가능해도 "목록 조회"만큼은 구조화된 JSON을
// 전혀 내려주지 않아(웹훅은 HTML 페이지, 권한은 해당 엔드포인트 자체가 없음) 스텁으로 남긴다.
func newAdminCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "관리자 운영 명령 (backup/webhook/permission)",
	}
	cmd.AddCommand(newAdminBackupCmd(ctx))
	cmd.AddCommand(newAdminWebhookCmd(ctx))
	cmd.AddCommand(newAdminPermissionCmd(ctx))
	return cmd
}

func newAdminBackupCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "사이트 전체 데이터 백업 다운로드/복원 (사이트 매니저 전용)",
	}
	cmd.AddCommand(newAdminBackupExportCmd(ctx))
	cmd.AddCommand(newAdminBackupImportCmd(ctx))
	return cmd
}

func newAdminBackupExportCmd(ctx *cmdContext) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "GET /site/export로 전체 데이터 백업 JSON을 내려받는다",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			data, err := client.ExportBackup(cmd.Context())
			if err != nil {
				return err
			}
			if output == "" || output == "-" {
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}
			if err := os.WriteFile(output, data, 0o600); err != nil {
				return fmt.Errorf("백업 파일을 쓸 수 없습니다(%s): %w", output, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "백업을 %s에 저장했습니다 (%d bytes).\n", output, len(data))
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "저장할 파일 경로 (생략 시 표준출력)")
	return cmd
}

func newAdminBackupImportCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <file>",
		Short: "POST /site/import로 백업 파일을 업로드해 전체 데이터를 복원한다",
		Long:  "주의: 서버의 모든 테이블을 업로드한 백업으로 완전히 교체한다(부분 복원이 아님).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file, err := os.Open(args[0])
			if err != nil {
				return fmt.Errorf("백업 파일을 열 수 없습니다(%s): %w", args[0], err)
			}
			defer file.Close()

			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			if err := client.ImportBackup(cmd.Context(), args[0], file); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "백업을 복원했습니다.")
			return nil
		},
	}
	return cmd
}

func newAdminWebhookCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "프로젝트 웹훅 생성/삭제 (목록 조회는 서버 API 부재로 미구현)",
	}
	cmd.AddCommand(newAdminWebhookCreateCmd(ctx))
	cmd.AddCommand(newAdminWebhookDeleteCmd(ctx))
	cmd.AddCommand(newAdminWebhookListCmd(ctx))
	return cmd
}

func newAdminWebhookCreateCmd(ctx *cmdContext) *cobra.Command {
	var repo, payloadURL, secret, webhookType string
	var gitPush bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "웹훅을 생성한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := parseRepo(repo)
			if err != nil {
				return err
			}
			if payloadURL == "" {
				return fmt.Errorf("--url은 필수입니다")
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			if err := client.CreateWebhook(cmd.Context(), owner, project, payloadURL, secret, gitPush, webhookType); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "웹훅을 생성했습니다.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (필수)`)
	cmd.Flags().StringVar(&payloadURL, "url", "", "Payload URL (필수)")
	cmd.Flags().StringVar(&secret, "secret", "", "Authorization Token(secret)")
	cmd.Flags().BoolVar(&gitPush, "git-push", false, "git push 이벤트에도 발송할지 여부")
	cmd.Flags().StringVar(&webhookType, "type", "SIMPLE", "웹훅 타입 (예: SIMPLE)")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}

func newAdminWebhookDeleteCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "웹훅을 삭제한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := parseRepo(repo)
			if err != nil {
				return err
			}
			id, err := parseNumberArg(args[0])
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			if err := client.DeleteWebhook(cmd.Context(), owner, project, id); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "웹훅을 삭제했습니다.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (필수)`)
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

// yona-wiki P3-02 Step9 조사 결과: 웹훅 목록은 web/WebhookController.kt의 GET 엔드포인트가
// Thymeleaf HTML 뷰(project/setting_webhook)만 반환해 CLI가 구조화된 데이터를 얻을 방법이
// 없다. 새 JSON API를 yuna 쪽에 추가하는 것은 이 CLI 작업의 범위 밖이라 미구현 스텁으로 둔다.
func newAdminWebhookListCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "미구현: 서버에 웹훅 목록 JSON API가 없다",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, _, err := parseRepo(repo); err != nil {
				return err
			}
			return errors.New(api.ErrNotSupportedByServer.Error() +
				" (웹훅 목록은 웹 UI에서 확인하세요: /projects/{owner}/{project}/webhooks)")
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (필수)`)
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func newAdminPermissionCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permission",
		Short: "프로젝트 멤버 권한 추가/변경/삭제 (목록 조회는 서버 API 부재로 미구현)",
	}
	cmd.AddCommand(newAdminPermissionAddCmd(ctx))
	cmd.AddCommand(newAdminPermissionUpdateRoleCmd(ctx))
	cmd.AddCommand(newAdminPermissionRemoveCmd(ctx))
	cmd.AddCommand(newAdminPermissionListCmd(ctx))
	return cmd
}

// resolveProjectID는 "owner/project" 표기를 web/ProjectMemberController.kt가 요구하는 숫자
// projectId로 변환한다(GET /api/v1/projects/{owner}/{project} 응답의 id 필드 재사용).
func resolveProjectID(ctx *cmdContext, cmd *cobra.Command, repo string) (client *api.Client, projectID int64, err error) {
	owner, project, err := parseRepo(repo)
	if err != nil {
		return nil, 0, err
	}
	client, err = ctx.newClient()
	if err != nil {
		return nil, 0, err
	}
	p, err := client.GetProject(cmd.Context(), owner, project)
	if err != nil {
		return nil, 0, fmt.Errorf("프로젝트 id를 조회할 수 없습니다: %w", err)
	}
	return client, p.ID, nil
}

func newAdminPermissionAddCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "add <loginId>",
		Short: "프로젝트 멤버를 추가한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, projectID, err := resolveProjectID(ctx, cmd, repo)
			if err != nil {
				return err
			}
			if _, err := client.AddProjectMember(cmd.Context(), projectID, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s님을 멤버로 추가했습니다.\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (필수)`)
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func newAdminPermissionUpdateRoleCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "update-role <userId> <roleId>",
		Short: "프로젝트 멤버의 역할을 변경한다",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, err := parseNumberArg(args[0])
			if err != nil {
				return err
			}
			roleID, err := parseNumberArg(args[1])
			if err != nil {
				return err
			}
			client, projectID, err := resolveProjectID(ctx, cmd, repo)
			if err != nil {
				return err
			}
			if _, err := client.UpdateProjectMemberRole(cmd.Context(), projectID, userID, roleID); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "역할을 변경했습니다.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (필수)`)
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func newAdminPermissionRemoveCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "remove <userId>",
		Short: "프로젝트 멤버를 제거한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, err := parseNumberArg(args[0])
			if err != nil {
				return err
			}
			client, projectID, err := resolveProjectID(ctx, cmd, repo)
			if err != nil {
				return err
			}
			if _, err := client.RemoveProjectMember(cmd.Context(), projectID, userID); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "멤버를 제거했습니다.")
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (필수)`)
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

// yona-wiki P3-02 Step9 조사 결과: web/ProjectMemberController.kt에는 "현재 멤버+역할" 목록을
// 내려주는 엔드포인트가 없다(가장 가까운 assignableUsers는 "할당 가능한 후보" 목록이지 이미
// 배정된 권한 매트릭스가 아니다). 새 API 추가는 범위 밖이라 미구현 스텁으로 둔다.
func newAdminPermissionListCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "미구현: 서버에 멤버/권한 목록 JSON API가 없다",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, _, err := parseRepo(repo); err != nil {
				return err
			}
			return errors.New(api.ErrNotSupportedByServer.Error() +
				" (프로젝트 설정 화면에서 확인하세요)")
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (필수)`)
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

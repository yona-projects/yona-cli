package cmd

import (
	"fmt"

	"github.com/search5/yona-cli/internal/api"
	"github.com/spf13/cobra"
)

// newTagCmd는 "git tag"/GitHub 저장소의 태그 관리에 대응 — yona-wiki P3-10이 추가한
// /api/v1/projects/{owner}/{project}/tags를 그대로 감싼다. gh CLI에는 독립된 최상위 "gh tag"가
// 없지만(태그는 gh release에 종속되거나 순수 git으로 다룸 — P3-10 계획 문서 조사 참고) yona는
// 릴리즈 개념이 없어 이 명령이 곧 태그 관리의 유일한 CLI 경로다.
func newTagCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "git 태그 조회/생성/삭제",
	}
	cmd.AddCommand(newTagListCmd(ctx))
	cmd.AddCommand(newTagCreateCmd(ctx))
	cmd.AddCommand(newTagDeleteCmd(ctx))
	return cmd
}

func newTagListCmd(ctx *cmdContext) *cobra.Command {
	var repo, jsonFields string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "태그 목록을 조회한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			tags, err := client.ListTags(cmd.Context(), owner, project)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, tags, jsonFields)
			}
			if len(tags) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "태그가 없습니다.")
				return nil
			}
			for _, t := range tags {
				kind := "lightweight"
				if annotated, ok := t["annotated"].(bool); ok && annotated {
					kind = "annotated"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", str(t, "name"), kind, str(t, "targetCommitId"))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json name,annotated,targetCommitId)")
	return cmd
}

func newTagCreateCmd(ctx *cmdContext) *cobra.Command {
	var repo, target, message string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "새 태그를 만든다(--message를 주면 annotated 태그, 생략하면 lightweight 태그)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			tag, err := client.CreateTag(cmd.Context(), owner, project, api.CreateTagRequest{
				Name:    args[0],
				Target:  target,
				Message: message,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "태그 %s 생성됨 (대상 커밋: %s)\n", str(tag, "name"), str(tag, "targetCommitId"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&target, "target", "", "태그를 붙일 커밋/브랜치 (생략하면 기본 브랜치의 HEAD)")
	cmd.Flags().StringVar(&message, "message", "", "annotated 태그 메시지 (생략하면 lightweight 태그가 만들어진다)")
	return cmd
}

func newTagDeleteCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "태그를 삭제한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			if err := client.DeleteTag(cmd.Context(), owner, project, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "태그 %s을(를) 삭제했습니다.\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	return cmd
}

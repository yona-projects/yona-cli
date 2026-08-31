package cmd

import (
	"fmt"

	"github.com/search5/yona-cli/internal/api"
	"github.com/spf13/cobra"
)

// newLabelCmd는 "gh label list/create/edit/delete" 대응 — yona-wiki P3-02 4라운드가 추가한
// /api/v1/projects/{owner}/{project}/labels를 그대로 감싼다.
func newLabelCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "label",
		Short: "프로젝트 라벨 조회/생성/수정/삭제",
	}
	cmd.AddCommand(newLabelListCmd(ctx))
	cmd.AddCommand(newLabelCreateCmd(ctx))
	cmd.AddCommand(newLabelEditCmd(ctx))
	cmd.AddCommand(newLabelDeleteCmd(ctx))
	return cmd
}

func newLabelListCmd(ctx *cmdContext) *cobra.Command {
	var repo, jsonFields string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "라벨 목록을 조회한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			labels, err := client.ListLabels(cmd.Context(), owner, project)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, labels, jsonFields)
			}
			if len(labels) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "라벨이 없습니다.")
				return nil
			}
			for _, l := range labels {
				fmt.Fprintf(cmd.OutOrStdout(), "#%s\t%s\n", num(l, "id"), str(l, "name"))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json id,name,color)")
	return cmd
}

func newLabelCreateCmd(ctx *cmdContext) *cobra.Command {
	var repo, name, color, category string
	var exclusive bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "새 라벨을 만든다",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			if name == "" || color == "" || category == "" {
				return fmt.Errorf("--name, --color, --category는 모두 필수입니다")
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			label, err := client.CreateLabel(cmd.Context(), owner, project, api.CreateLabelRequest{
				Name: name, Color: color, Category: category, CategoryIsExclusive: exclusive,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "라벨 #%s 생성됨: %s\n", num(label, "id"), str(label, "name"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&name, "name", "", "라벨 이름 (필수)")
	cmd.Flags().StringVar(&color, "color", "", "라벨 색상 (필수)")
	cmd.Flags().StringVar(&category, "category", "", "라벨 카테고리 (필수)")
	cmd.Flags().BoolVar(&exclusive, "exclusive", false, "카테고리 내 배타적 선택 여부")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("color")
	_ = cmd.MarkFlagRequired("category")
	return cmd
}

func newLabelEditCmd(ctx *cmdContext) *cobra.Command {
	var repo, name, color string
	var categoryID int64
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "라벨을 수정한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
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
			// TASK-0421(yona-wiki P3-02 10라운드) — 라벨 수정 엔드포인트
			// (LabelRestApiController.update -> ProjectViewController.updateLabelForm 위임) 응답
			// 바디에 "id" 필드가 없어(수정된 라벨 객체 자체가 불완전하게 내려옴) num(label, "id")가
			// 항상 "-"를 반환해 "라벨 #-을(를) 수정했습니다."로 깨져 나왔다. 서버 응답 형식에
			// 의존하지 않고 사용자가 입력한 args[0](id)을 그대로 메시지에 쓰는 게 더 간단하고
			// 안전하다 — 애초에 서버가 뭘 내려주든 사용자가 수정을 요청한 라벨 id는 이미 알고 있다.
			if _, err := client.UpdateLabel(cmd.Context(), owner, project, id, api.UpdateLabelRequest{
				Name: name, Color: color, CategoryID: categoryID,
			}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "라벨 #%s을(를) 수정했습니다.\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&name, "name", "", "새 이름 (필수)")
	cmd.Flags().StringVar(&color, "color", "", "새 색상 (필수)")
	cmd.Flags().Int64Var(&categoryID, "category-id", 0, "카테고리 숫자 ID (필수)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("color")
	_ = cmd.MarkFlagRequired("category-id")
	return cmd
}

func newLabelDeleteCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "라벨을 삭제한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
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
			if err := client.DeleteLabel(cmd.Context(), owner, project, id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "라벨 #%d을(를) 삭제했습니다.\n", id)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	return cmd
}

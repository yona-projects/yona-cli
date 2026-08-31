package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newOrgCmd는 "gh org" 대응 — yona-wiki P3-02 4라운드가 추가한
// /api/v1/organizations, /api/v1/organizations/{name}을 그대로 감싼다.
func newOrgCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "조직 목록/조회",
	}
	cmd.AddCommand(newOrgListCmd(ctx))
	cmd.AddCommand(newOrgViewCmd(ctx))
	return cmd
}

func newOrgListCmd(ctx *cmdContext) *cobra.Command {
	var filter string
	var page int
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "조직 목록을 조회한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			result, err := client.ListOrganizations(cmd.Context(), filter, page)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, result.Content, jsonFields)
			}
			if len(result.Content) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "조직이 없습니다.")
				return nil
			}
			for _, org := range result.Content {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", org.Name, org.Descr)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&filter, "filter", "", "이름/설명에 대한 부분 검색어")
	cmd.Flags().IntVar(&page, "page", 0, "페이지 번호 (0부터 시작)")
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json name,descr)")
	return cmd
}

func newOrgViewCmd(ctx *cmdContext) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "view <name>",
		Short: "조직 하나를 조회한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			org, err := client.GetOrganization(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, org, jsonFields)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\n%s\n\n소속 프로젝트:\n", org.Name, org.Descr)
			for _, p := range org.Projects {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s/%s\n", p.Owner, p.Name)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json name,descr,projects)")
	return cmd
}

package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/yona-projects/yona-cli/internal/config"
)

// newAliasCmd는 "gh alias set/list/delete" 대응이다. gh는 "!"로 시작하는 셸 명령 별칭도
// 지원하지만, 그건 저장된 문자열을 셸로 그대로 넘겨 실행한다는 뜻이라 config.yml을 편집할 수
// 있는 사람이 임의 명령을 실행시킬 수 있는 경로가 된다 — yona-cli는 그 위험을 피하려고 다른
// yona 하위 명령으로만 확장되는 별칭만 지원한다(expandAlias, root.go 참고).
func newAliasCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alias",
		Short: "자주 쓰는 명령에 별칭을 만든다",
	}
	cmd.AddCommand(newAliasSetCmd())
	cmd.AddCommand(newAliasListCmd())
	cmd.AddCommand(newAliasDeleteCmd())
	return cmd
}

func newAliasSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <name> <expansion>",
		Short: `별칭을 등록한다 (예: yona alias set pv "pr view")`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, expansion := args[0], args[1]
			if isBuiltinCommand(cmd.Root(), name) {
				return fmt.Errorf("%q는 이미 내장 명령이라 별칭으로 쓸 수 없습니다", name)
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.SetAlias(name, expansion)
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", name, expansion)
			return nil
		},
	}
}

func newAliasListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "등록된 별칭 목록을 보여준다",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			names := make([]string, 0, len(cfg.Aliases))
			for name := range cfg.Aliases {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", name, cfg.Aliases[name])
			}
			return nil
		},
	}
}

func newAliasDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "별칭을 삭제한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if !cfg.RemoveAlias(args[0]) {
				return fmt.Errorf("등록되지 않은 별칭입니다: %s", args[0])
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s 별칭을 삭제했습니다.\n", args[0])
			return nil
		},
	}
}

// isBuiltinCommand는 name이 root에 이미 등록된 최상위 명령(또는 그 별칭)인지 확인한다.
func isBuiltinCommand(root *cobra.Command, name string) bool {
	for _, c := range root.Commands() {
		if c.Name() == name || c.HasAlias(name) {
			return true
		}
	}
	return false
}

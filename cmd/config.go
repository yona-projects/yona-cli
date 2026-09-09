package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yona-projects/yona-cli/internal/config"
)

// configKeys는 "yona config get/set/list"가 다루는 설정 키와 설명이다. gh CLI처럼 정해진
// 키만 허용한다(오타를 조용히 저장해버리지 않도록) — 새 설정을 추가하려면 여기에 키만
// 등록하면 get/set/list가 자동으로 반영한다.
var configKeys = map[string]string{
	"browser": "--web/browse 계열이 열 브라우저 명령 (예: chromium) — 비우면 OS 기본 브라우저(BROWSER 환경변수가 우선)",
}

// newConfigCmd는 "gh config get/set/list" 대응이다.
func newConfigCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "yona-cli 로컬 설정값을 관리한다",
	}
	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigListCmd())
	return cmd
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "설정값을 조회한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireKnownConfigKey(args[0]); err != nil {
				return err
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			value, _ := cfg.GetSetting(args[0])
			fmt.Fprintln(cmd.OutOrStdout(), value)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "설정값을 저장한다",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireKnownConfigKey(args[0]); err != nil {
				return err
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.SetSetting(args[0], args[1])
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s = %s\n", args[0], args[1])
			return nil
		},
	}
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "설정 가능한 키와 현재 값을 보여준다",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			for _, key := range sortedConfigKeys() {
				value, _ := cfg.GetSetting(key)
				fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", key, value)
			}
			return nil
		},
	}
}

func requireKnownConfigKey(key string) error {
	if _, ok := configKeys[key]; !ok {
		return fmt.Errorf("알 수 없는 설정 키입니다: %s (지원 키: %s)", key, strings.Join(sortedConfigKeys(), ", "))
	}
	return nil
}

func sortedConfigKeys() []string {
	keys := make([]string, 0, len(configKeys))
	for key := range configKeys {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

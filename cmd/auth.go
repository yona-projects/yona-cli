package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/yona-projects/yona-cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newAuthCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "yona 서버 인증 관리 (login/logout/status)",
	}
	cmd.AddCommand(newAuthLoginCmd(ctx))
	cmd.AddCommand(newAuthLogoutCmd(ctx))
	cmd.AddCommand(newAuthStatusCmd(ctx))
	return cmd
}

// yona-wiki P3-02 "CLI 로그인 토큰의 기본 스코프"(2026-08-28 결정) 대응: yona auth login은
// 사용자 본인 계정의 전체 권한을 가진 토큰(레거시 전권 토큰 또는 전체 스코프로 발급한
// Fine-grained 토큰)을 그대로 저장하는 것을 기본 흐름으로 삼는다 — yuna 서버에 "로그인 API"
// 자체가 없고(세션 로그인은 브라우저 폼 기반, 토큰은 웹 UI에서 발급) OAuth 유사 플로우도 없으므로
// gh CLI의 `gh auth login --with-token`과 동일하게 "이미 발급받은 토큰 값을 CLI에 알려주는"
// 방식으로 구현한다. 제한된 토큰을 쓰고 싶으면 --token으로 그 값을 직접 넘기면 된다(범위
// 제한은 토큰 발급 시점에 웹 UI에서 이미 결정돼 있으므로 CLI가 별도로 스코프를 선택하지 않는다).
func newAuthLoginCmd(ctx *cmdContext) *cobra.Command {
	var tokenFlag string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Personal Access Token으로 yona 서버에 로그인한다",
		Long: "yona 서버 URL과 Personal Access Token을 로컬 설정 파일에 저장한다.\n" +
			"토큰은 yona 웹 UI(사용자 설정 > API 토큰)에서 미리 발급받아야 한다.\n" +
			"--token을 생략하면 표준입력에서 프롬프트로 입력받는다.",
		RunE: func(cmd *cobra.Command, args []string) error {
			server := *ctx.server
			if server == "" {
				server = os.Getenv(config.ServerEnvVar)
			}
			if server == "" {
				return fmt.Errorf("서버 URL이 필요합니다 — --server 플래그 또는 %s 환경변수를 지정하세요", config.ServerEnvVar)
			}

			token := tokenFlag
			if token == "" {
				var err error
				token, err = promptForToken(cmd)
				if err != nil {
					return err
				}
			}
			if strings.TrimSpace(token) == "" {
				return fmt.Errorf("토큰이 비어 있습니다")
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.SetHost(server, strings.TrimSpace(token))
			if err := config.Save(cfg); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%s 서버에 로그인했습니다.\n", server)
			return nil
		},
	}
	cmd.Flags().StringVar(&tokenFlag, "token", "", "저장할 Personal Access Token (생략 시 프롬프트로 입력)")
	return cmd
}

// promptForToken은 표준입력이 실제 터미널(TTY)이면 golang.org/x/term으로 입력을 화면에 표시하지
// 않고 받는다. 테스트나 파이프 입력처럼 TTY가 아닌 경우(cmd.InOrStdin()이 일반 io.Reader일 때)는
// 한 줄을 그대로 읽는다.
func promptForToken(cmd *cobra.Command) (string, error) {
	fmt.Fprint(cmd.OutOrStdout(), "Personal Access Token: ")

	if f, ok := cmd.InOrStdin().(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		data, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(cmd.OutOrStdout())
		if err != nil {
			return "", fmt.Errorf("토큰을 입력받을 수 없습니다: %w", err)
		}
		return string(data), nil
	}

	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("토큰을 입력받을 수 없습니다: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func newAuthLogoutCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "저장된 로그인 정보를 삭제한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			server, err := config.ResolveServer(*ctx.server)
			if err != nil {
				return err
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Hosts[server]; !ok {
				return fmt.Errorf("%s 서버에 로그인된 정보가 없습니다", server)
			}
			cfg.RemoveHost(server)
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s 서버에서 로그아웃했습니다.\n", server)
			return nil
		},
	}
	return cmd
}

func newAuthStatusCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "현재 로그인 상태를 표시한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if len(cfg.Hosts) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "로그인된 yona 서버가 없습니다. 'yona auth login'을 실행하세요.")
				return nil
			}

			for server, host := range cfg.Hosts {
				marker := "  "
				if server == cfg.CurrentHost {
					marker = "* "
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", marker, server)
				fmt.Fprintf(cmd.OutOrStdout(), "    토큰: %s\n", maskToken(host.Token))
			}
			return nil
		},
	}
	return cmd
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "********"
	}
	return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}

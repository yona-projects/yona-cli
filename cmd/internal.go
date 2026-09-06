// yona-wiki P3-03 Step4 — 리눅스/맥 시스템 OpenSSH의 AuthorizedKeysCommand 훅으로 실행되는 두
// 서브커맨드. 일반 사용자가 직접 실행할 일이 없으므로(Hidden: true) `yona --help`에는 나타나지
// 않는다.
//
// sshd_config 설정 예시(실제 등록은 이 자동화 세션의 범위 밖 — docs/yona-wiki/plans/
// p3-03-ssh-gpg.md의 "호스트 시스템 제약" 참고):
//
//	AuthorizedKeysCommand /usr/local/bin/yona internal ssh-auth %t %k
//	AuthorizedKeysCommandUser nobody
//
// ssh-auth가 표준출력으로 내보내는 authorized_keys 한 줄에 forced command로 ssh-shell이
// 박혀 있어(BuildAuthorizedKeysLine), 실제 git 클라이언트가 접속하면 sshd가 클라이언트가 보낸
// SSH_ORIGINAL_COMMAND(git-upload-pack/git-receive-pack)를 무시하고 이 forced command를
// 실행한다 — ssh-shell이 그 원본 명령을 SSH_ORIGINAL_COMMAND 환경변수로 다시 읽어 인가를
// 재확인한 뒤 실제 git 프로세스를 exec한다.
package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/search5/yona-cli/internal/sshhelper"
	"github.com/spf13/cobra"
)

func newInternalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "internal",
		Short:  "내부 전용 명령(시스템 sshd 훅 전용, 직접 실행하지 마세요)",
		Hidden: true,
	}
	cmd.AddCommand(newInternalSshAuthCmd())
	cmd.AddCommand(newInternalSshShellCmd())
	return cmd
}

// BuildAuthorizedKeysLine은 authorized_keys 한 줄(옵션 + 키 타입 + base64)을 만든다. 순수
// 문자열 조립 함수라 유닛테스트로 직접 검증한다(internal_test.go).
//
// no-port-forwarding/no-X11-forwarding/no-agent-forwarding/no-pty는 GitHub/GitLab의 Deploy
// Key·서비스 계정 SSH 키가 공통으로 쓰는 옵션과 동일하다 — 이 키로는 git 프로토콜 외의 어떤
// SSH 기능도 쓸 수 없게 막는다.
func BuildAuthorizedKeysLine(selfPath, principal, keyType, keyBase64 string) string {
	return fmt.Sprintf(
		`command="%s internal ssh-shell --principal=%s",no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty %s %s`,
		selfPath, principal, keyType, keyBase64,
	)
}

func newInternalSshAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "ssh-auth <key-type> <key-base64>",
		Short:  "AuthorizedKeysCommand 훅 — 공개키를 조회해 forced command가 박힌 authorized_keys 한 줄을 출력한다",
		Hidden: true,
		Args:   cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyType, keyBase64 := args[0], args[1]

			cfg, err := sshhelper.LoadConfig()
			if err != nil {
				return err
			}
			client := sshhelper.NewClient(cfg)

			publicKeyLine := keyType + " " + keyBase64
			resp, statusCode, err := client.Authenticate(cmd.Context(), publicKeyLine)
			if err != nil {
				return err
			}

			// 404(알 수 없는 공개키)는 "이 훅은 이 키를 모른다"는 정상적인 결과다 — 아무것도
			// 출력하지 않으면 sshd가 이 authorized_keys 후보를 무시하고 계속 진행한다(다른
			// AuthorizedKeysFile/훅으로). 종료 코드는 0으로 유지한다 — 비정상 상황이 아니다.
			if statusCode == http.StatusNotFound {
				return nil
			}
			if statusCode != http.StatusOK {
				return fmt.Errorf("ssh-auth: 서버가 예기치 않은 상태코드를 반환했습니다: %d", statusCode)
			}

			selfPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("실행 파일 경로를 확인할 수 없습니다: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), BuildAuthorizedKeysLine(selfPath, resp.Principal, keyType, keyBase64))
			return nil
		},
	}
	return cmd
}

// ParseSshOriginalCommand는 SSH_ORIGINAL_COMMAND 값("git-upload-pack '/owner/project.git'"
// 형태)이 알려진 git 서비스 명령인지만 얕게 검사한다 — 실제 인가 판정(스코프/read_only 등)은
// 서버(SshAuthService.authorizeGitCommand)가 하므로, 이 CLI는 완전히 다른 형식의 명령(예:
// 대화형 셸 시도)을 조기에 거부하는 정도의 방어만 한다.
func ParseSshOriginalCommand(command string) error {
	trimmed := command
	for _, prefix := range []string{"git-upload-pack ", "git-receive-pack ", "git-upload-archive "} {
		if len(trimmed) >= len(prefix) && trimmed[:len(prefix)] == prefix {
			return nil
		}
	}
	return fmt.Errorf("지원하지 않는 명령입니다: %s", command)
}

func newInternalSshShellCmd() *cobra.Command {
	var principal string
	cmd := &cobra.Command{
		Use:    "ssh-shell",
		Short:  "forced command — SSH_ORIGINAL_COMMAND를 인가 확인 후 실제 git 프로세스로 exec한다",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			originalCommand := os.Getenv("SSH_ORIGINAL_COMMAND")
			if originalCommand == "" {
				return fmt.Errorf("ssh-shell: 대화형 셸은 지원하지 않습니다(SSH_ORIGINAL_COMMAND 없음)")
			}
			if err := ParseSshOriginalCommand(originalCommand); err != nil {
				return err
			}

			cfg, err := sshhelper.LoadConfig()
			if err != nil {
				return err
			}
			client := sshhelper.NewClient(cfg)

			authz, err := client.Authorize(cmd.Context(), principal, originalCommand)
			if err != nil {
				return err
			}
			if !authz.Allowed {
				fmt.Fprintln(cmd.ErrOrStderr(), authz.Reason)
				return fmt.Errorf("접근이 거부되었습니다")
			}

			return RunGitService(authz.Service, authz.RepoDir, cmd)
		},
	}
	cmd.Flags().StringVar(&principal, "principal", "", "ssh-auth가 authenticate 단계에서 발급한 opaque principal")
	_ = cmd.MarkFlagRequired("principal")
	return cmd
}

// RunGitService는 서버가 승인한 서비스(git-upload-pack/git-receive-pack)를 실제 시스템 git
// 바이너리로 exec하고, 완료될 때까지 기다린다. stdin/stdout/stderr을 그대로 이어붙여야
// 클라이언트와의 git wire 프로토콜 교환이 정상 동작한다.
func RunGitService(service, repoDir string, cmd *cobra.Command) error {
	switch service {
	case "git-upload-pack", "git-receive-pack":
		gitCmd := exec.CommandContext(cmd.Context(), service, repoDir)
		gitCmd.Stdin = cmd.InOrStdin()
		gitCmd.Stdout = cmd.OutOrStdout()
		gitCmd.Stderr = cmd.ErrOrStderr()
		return gitCmd.Run()
	default:
		return fmt.Errorf("지원하지 않는 서비스입니다: %s", service)
	}
}

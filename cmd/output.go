package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// parseNumberArg는 이슈/PR 번호 인자를 파싱한다.
func parseNumberArg(arg string) (int64, error) {
	n, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("번호는 숫자여야 합니다 (입력값: %q)", arg)
	}
	return n, nil
}

// parseRepo는 gh CLI의 "-R owner/repo" 관례를 그대로 따른 "owner/project" 표기를 나눈다.
func parseRepo(repo string) (owner string, project string, err error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf(`--repo 값은 "owner/project" 형식이어야 합니다 (입력값: %q)`, repo)
	}
	return parts[0], parts[1], nil
}

// printJSON은 --json 플래그가 켜졌을 때 원본 응답 구조를 그대로 예쁘게 출력한다. Issue/PullRequest
// 응답은 map[string]interface{}로 느슨하게 받으므로, 서버가 실제로 어떤 필드를 내려주는지
// 있는 그대로 보고 싶을 때 이 경로가 유일한 정확한 수단이다.
func printJSON(cmd *cobra.Command, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON으로 변환할 수 없습니다: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

// str/num은 JPA 엔티티 직접 직렬화 응답(map[string]interface{})에서 흔한 키를 방어적으로
// 꺼낸다 — 키가 없거나 타입이 다르면 자리표시자를 돌려줄 뿐 패닉하지 않는다(서버 응답 스키마가
// CLI 배포와 독립적으로 바뀔 수 있음을 전제).
func str(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return "-"
}

func num(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return fmt.Sprintf("%d", int64(n))
		default:
			return fmt.Sprintf("%v", n)
		}
	}
	return "-"
}

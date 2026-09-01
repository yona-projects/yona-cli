package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/search5/yona-cli/internal/gitutil"
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

// resolveRepo는 --repo/-R 플래그 값이 주어지면 그것을 파싱하고, 비어 있으면 현재 디렉터리의 git
// origin remote로부터 owner/project를 자동감지한다(gh CLI가 "github.com/owner/repo"를 파싱하는
// 것과 동일한 관례 — yuna clone URL 형식은 TemplateHelper.getCloneUrl() 참고). git이 없거나
// 저장소 밖이거나 origin이 없으면 "--repo 필수" 오류로 폴백한다.
func resolveRepo(cmd *cobra.Command, repo string) (owner string, project string, err error) {
	if repo != "" {
		return parseRepo(repo)
	}
	owner, project, err = gitutil.DetectRepo(cmd.Context())
	if err != nil {
		return "", "", fmt.Errorf("--repo(-R) 값이 필요합니다 — 현재 디렉터리에서 대상 프로젝트를 자동감지할 수 없습니다"+
			"(git 저장소가 아니거나 origin remote가 없음): %w", err)
	}
	return owner, project, nil
}

// printJSON은 --json <fields> 플래그로 지정한 필드만(콤마 구분) 뽑아 JSON으로 출력한다(gh CLI의
// "--json number,title,state" 관례 — cmd/output.go의 예전 불리언 스위치를 대체). v가 배열/슬라이스면
// 각 원소마다 같은 필드를 뽑고, 단일 객체면 그 객체에서만 뽑는다. 존재하지 않는 필드명은 조용히
// 생략한다(오탈자를 쳐도 패닉하지 않고 그 키가 결과에 안 나올 뿐이다).
func printJSON(cmd *cobra.Command, v interface{}, fields string) error {
	fieldList := splitFields(fields)
	if len(fieldList) == 0 {
		return fmt.Errorf(`--json 플래그는 콤마로 구분한 필드 목록이 필요합니다 (예: --json number,title,state)`)
	}

	generic, err := toGenericJSON(v)
	if err != nil {
		return err
	}
	filtered := pickFields(generic, fieldList)

	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON으로 변환할 수 없습니다: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

func splitFields(fields string) []string {
	var out []string
	for _, f := range strings.Split(fields, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

func toGenericJSON(v interface{}) (interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("JSON으로 변환할 수 없습니다: %w", err)
	}
	var generic interface{}
	if err := json.Unmarshal(data, &generic); err != nil {
		return nil, fmt.Errorf("JSON을 해석할 수 없습니다: %w", err)
	}
	return generic, nil
}

func pickFields(v interface{}, fields []string) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		out := map[string]interface{}{}
		for _, f := range fields {
			if val, ok := t[f]; ok {
				out[f] = val
			}
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, item := range t {
			out[i] = pickFields(item, fields)
		}
		return out
	default:
		return v
	}
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

// rawStr은 str()과 달리 값이 없을 때 "-" 자리표시자 대신 빈 문자열을 돌려준다 — read-modify-write
// 방식으로 값을 재전송해야 하는 edit류 명령(예: issue edit, pr edit)에서 "-"가 실제 값으로 오전송되는
// 사고를 막기 위해 쓴다.
func rawStr(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
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

// rawInt64는 map[string]interface{}에서 숫자 필드를 int64로 꺼낸다(ok=false면 없거나 null).
// yona-wiki P3-02 13라운드(TASK-0430) — `issue edit`/`pr edit`가 read-modify-write로 현재
// milestoneId/assignee.userId/labels[].id를 재전송해 보존해야 할 때 쓴다.
func rawInt64(m map[string]interface{}, key string) (int64, bool) {
	if v, ok := m[key]; ok && v != nil {
		if f, ok := v.(float64); ok {
			return int64(f), true
		}
	}
	return 0, false
}

// rawMap은 map[string]interface{}에서 중첩된 객체 필드를 꺼낸다(예: issue["assignee"]).
func rawMap(m map[string]interface{}, key string) (map[string]interface{}, bool) {
	if v, ok := m[key]; ok && v != nil {
		if mm, ok := v.(map[string]interface{}); ok {
			return mm, true
		}
	}
	return nil, false
}

// rawSlice는 map[string]interface{}에서 중첩된 배열 필드를 꺼낸다(예: issue["labels"]).
func rawSlice(m map[string]interface{}, key string) []interface{} {
	if v, ok := m[key]; ok && v != nil {
		if s, ok := v.([]interface{}); ok {
			return s
		}
	}
	return nil
}

// currentLabelIDs는 issue/PR 응답의 labels 배열(각 원소가 {"id": ..., ...})에서 id만 뽑는다.
func currentLabelIDs(m map[string]interface{}) []int64 {
	var ids []int64
	for _, item := range rawSlice(m, "labels") {
		if lm, ok := item.(map[string]interface{}); ok {
			if id, ok := rawInt64(lm, "id"); ok {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// currentAssigneeUserID는 issue/PR 응답의 assignee 객체({"userId": ...})에서 userId를 뽑는다.
func currentAssigneeUserID(m map[string]interface{}) (int64, bool) {
	if am, ok := rawMap(m, "assignee"); ok {
		return rawInt64(am, "userId")
	}
	return 0, false
}

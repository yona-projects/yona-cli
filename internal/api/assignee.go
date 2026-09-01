package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// AssignableUser는 web/ProjectMemberController.kt.assignableUsers()가 내려주는 담당자 후보 한 명을
// 담는다. yona-wiki P3-02 13라운드(TASK-0430)가 이 엔드포인트에 userId 필드를 추가하기 전에는
// loginId만 내려줘서 issue/PR REST API가 요구하는 숫자 assigneeId로 변환할 방법이 아예 없었다.
type AssignableUser struct {
	UserID  int64  `json:"userId"`
	LoginID string `json:"loginId"`
	Name    string `json:"name"`
}

// ListAssignableUsers는 GET /api/projects/{projectId}/assignableUsers[?query=]를 호출한다.
// projectId는 숫자 ID다(web/ProjectMemberController.kt의 레거시 경로 그대로).
func (c *Client) ListAssignableUsers(ctx context.Context, projectID int64, query string) ([]AssignableUser, error) {
	path := fmt.Sprintf("/api/projects/%d/assignableUsers", projectID)
	if query != "" {
		path += "?" + url.Values{"query": {query}}.Encode()
	}
	var out []AssignableUser
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ResolveAssigneeID는 "owner/project" 프로젝트에서 loginId와 정확히 일치하는 담당자 후보를 찾아
// 그 userId를 돌려준다. issue/PR create·edit REST API는 담당자를 숫자 assigneeId/userId로만
// 받으므로(로그인ID를 직접 받지 않음) CLI가 사람이 입력하는 loginId를 이 값으로 변환해야 한다
// (yona-wiki P3-02 13라운드/TASK-0430 — `issue create/edit --assignee`, `pr edit --assignee`).
func (c *Client) ResolveAssigneeID(ctx context.Context, owner, project, loginID string) (int64, error) {
	p, err := c.GetProject(ctx, owner, project)
	if err != nil {
		return 0, fmt.Errorf("프로젝트 id를 조회할 수 없습니다: %w", err)
	}
	candidates, err := c.ListAssignableUsers(ctx, p.ID, loginID)
	if err != nil {
		return 0, err
	}
	for _, cand := range candidates {
		if cand.LoginID == loginID {
			return cand.UserID, nil
		}
	}
	return 0, fmt.Errorf("담당자로 지정할 수 있는 후보 중 로그인ID %q를 찾을 수 없습니다(프로젝트 멤버가 아닐 수 있습니다)", loginID)
}

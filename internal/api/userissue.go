package api

import (
	"context"
	"net/http"
)

// IssueStatusGroup은 UserIssueStatusRestApiController.kt가 assigned/created 각각에 대해
// 내려주는 {openCount, closedCount, items} 구조와 일치한다. items는 Issue 엔티티 목록이라
// map으로 느슨하게 받는다.
type IssueStatusGroup struct {
	OpenCount   int64                    `json:"openCount"`
	ClosedCount int64                    `json:"closedCount"`
	Items       []map[string]interface{} `json:"items"`
}

// IssueStatus는 GET /api/v1/user/issues/status의 응답 필드와 일치한다("gh issue status"의
// 최소 버전 — 담당/작성 이슈 개수·목록만 제공한다).
type IssueStatus struct {
	Assigned IssueStatusGroup `json:"assigned"`
	Created  IssueStatusGroup `json:"created"`
}

// GetIssueStatus는 GET /api/v1/user/issues/status를 호출한다. 이 엔드포인트는 세션 로그인/
// 레거시 전권 토큰으로만 인증되고(계획 문서에 기록된 스코프 인가 갭 — 저장소 비종속 전역
// 엔드포인트), Fine-grained 스코프 토큰으로는 401이 반환될 수 있다.
func (c *Client) GetIssueStatus(ctx context.Context) (*IssueStatus, error) {
	var out IssueStatus
	if err := c.DoJSON(ctx, http.MethodGet, "/api/v1/user/issues/status", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

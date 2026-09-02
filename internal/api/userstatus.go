package api

import (
	"context"
	"net/http"
)

// StatusSection은 UserStatusRestApiController.StatusSectionResponse({openCount, closedCount,
// items})와 일치한다. items는 IssueResponse/PullRequestResponse 목록이라 map으로 느슨하게 받는다
// (IssueStatusGroup과 동일한 관례).
type StatusSection struct {
	OpenCount   int64                    `json:"openCount"`
	ClosedCount int64                    `json:"closedCount"`
	Items       []map[string]interface{} `json:"items"`
}

// RepositoryActivityItem은 UserStatusRestApiController.RepositoryActivityItemResponse와 일치한다.
type RepositoryActivityItem struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	EventType    string `json:"eventType"`
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	SenderID     *int64 `json:"senderId"`
	Created      string `json:"created"`
}

// UserStatus는 GET /api/v1/user/status의 응답 필드와 일치한다("gh status" 대응 — 이미 있던
// GetIssueStatus(/api/v1/user/issues/status)보다 범위가 넓어 PR(담당/리뷰요청)과 저장소 활동까지
// 포함한다). Mentions 섹션은 yuna의 MentionService가 이슈(ISSUE_POST/ISSUE_COMMENT)만 추적하고
// PR 본문·리뷰 코멘트 멘션은 감지하지 않아 "멘션된 이슈"만 담는다(서버 쪽 주석 참고).
type UserStatus struct {
	AssignedIssues       StatusSection            `json:"assignedIssues"`
	AssignedPullRequests StatusSection            `json:"assignedPullRequests"`
	ReviewRequests       StatusSection            `json:"reviewRequests"`
	MentionedIssues      StatusSection            `json:"mentionedIssues"`
	RepositoryActivity   []RepositoryActivityItem `json:"repositoryActivity"`
}

// GetUserStatus는 GET /api/v1/user/status를 호출한다. GetIssueStatus와 마찬가지로 세션 로그인/
// 레거시 전권 토큰 또는 ISSUES+PULL_REQUESTS 두 스코프를 모두 가진 Fine-grained PAT으로 인증된다
// (서버 쪽 ApiTokenAuthenticationFilter.userStatusApiPattern 참고 — 두 스코프 중 하나라도 없으면
// 403).
func (c *Client) GetUserStatus(ctx context.Context) (*UserStatus, error) {
	var out UserStatus
	if err := c.DoJSON(ctx, http.MethodGet, "/api/v1/user/status", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

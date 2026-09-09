package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// IssueStatusGroup은 UserIssueStatusRestApiController.kt의 sectionNode()가 각 섹션마다
// 내려주는 {openCount, closedCount, items, totalElements, totalPages, page} 구조와 일치한다.
// items는 Issue 엔티티 목록이라 map으로 느슨하게 받는다.
type IssueStatusGroup struct {
	OpenCount     int64                    `json:"openCount"`
	ClosedCount   int64                    `json:"closedCount"`
	Items         []map[string]interface{} `json:"items"`
	TotalElements int64                    `json:"totalElements"`
	TotalPages    int64                    `json:"totalPages"`
	Page          int64                    `json:"page"`
}

// IssueStatus는 GET /api/v1/user/issues/status의 응답 필드와 일치한다("gh issue status" 대응 —
// 담당(assigned)/작성(created) 이슈뿐 아니라 댓글단(commented)/멘션된(mentioned)/
// 즐겨찾기한(favorite)/공유받은(shared) 이슈까지 6개 섹션을 제공한다).
type IssueStatus struct {
	Assigned  IssueStatusGroup `json:"assigned"`
	Created   IssueStatusGroup `json:"created"`
	Commented IssueStatusGroup `json:"commented"`
	Mentioned IssueStatusGroup `json:"mentioned"`
	Favorite  IssueStatusGroup `json:"favorite"`
	Shared    IssueStatusGroup `json:"shared"`
}

// IssueStatusOptions는 GET /api/v1/user/issues/status의 필터/페이지네이션 파라미터다. 값이
// 비어있는(zero value) 필드는 쿼리에 넣지 않고 서버 기본값(state=open, pageNum=1)을 그대로
// 쓴다. commenterId/mentionId/sharerId/favoriteId(다른 사용자 기준 조회)는 이 CLI에서는
// 노출하지 않는다 — "내 이슈 현황"이라는 명령 성격상 항상 로그인한 사용자 자신을 본다.
type IssueStatusOptions struct {
	State  string // open(기본값)/closed/all
	Filter string // 제목/본문 키워드 검색
	Page   int    // 1부터 시작, 생략(0) 시 서버 기본값(1)
}

// GetIssueStatus는 GET /api/v1/user/issues/status를 호출한다. `/api/v1/user/issues/**`는
// ApiTokenAuthenticationFilter.userApiPattern으로 ISSUES 스코프의 Fine-grained 토큰도
// 인증되도록 서버 쪽에서 이미 확장돼 있다(project는 null로 두고 계정 수준으로 인가).
func (c *Client) GetIssueStatus(ctx context.Context, opts IssueStatusOptions) (*IssueStatus, error) {
	path := "/api/v1/user/issues/status"
	q := url.Values{}
	if opts.State != "" {
		q.Set("state", opts.State)
	}
	if opts.Filter != "" {
		q.Set("filter", opts.Filter)
	}
	if opts.Page > 0 {
		q.Set("pageNum", strconv.Itoa(opts.Page))
	}
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var out IssueStatus
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// CreatePullRequestRequest는 web/PullRequestController.kt의 CreatePullRequestRequest와 필드가
// 동일해야 한다(PullRequestApiController.create()가 그대로 역직렬화해 위임한다).
type CreatePullRequestRequest struct {
	Title         string `json:"title"`
	Body          string `json:"body,omitempty"`
	FromProjectID int64  `json:"fromProjectId"`
	FromBranch    string `json:"fromBranch"`
	ToBranch      string `json:"toBranch"`
}

// UpdatePullRequestRequest는 web/PullRequestController.kt의 UpdatePullRequestRequest와 필드가
// 동일해야 한다. title은 non-null 필수, 나머지는 null이면 서버가 기존 값을 유지한다.
type UpdatePullRequestRequest struct {
	Title      string  `json:"title"`
	Body       *string `json:"body,omitempty"`
	FromBranch *string `json:"fromBranch,omitempty"`
	ToBranch   *string `json:"toBranch,omitempty"`
}

// PullRequestCommentRequest는 web/PullRequestController.kt의 PullRequestCommentRequest와 필드가
// 동일해야 한다.
type PullRequestCommentRequest struct {
	Body string `json:"body"`
}

// PullRequestListOptions는 GET .../pull-requests의 선택 쿼리 파라미터를 담는다. Limit은 서버가
// 이 엔드포인트에 페이지네이션을 지원하지 않아(List<PullRequest> 그대로 반환) 클라이언트 사이드
// 슬라이싱으로 처리한다 — cmd 계층에서 응답을 받은 뒤 자른다.
//
// yona-wiki P3-02 7라운드가 PR에 assignee/label 개념을 신설하면서 목록 필터도 함께 추가했지만
// (PullRequestController.getPullRequests()의 assignee/label 파라미터) CLI 쪽 배선이 없었다 —
// 13라운드(TASK-0430)가 해소. Assignee는 로그인ID, Label은 라벨 이름 등가비교다(서버 그대로).
type PullRequestListOptions struct {
	State    string
	Author   string
	Assignee string
	Label    string
}

func pullRequestsBasePath(owner, project string) string {
	return fmt.Sprintf("/api/v1/projects/%s/%s/pull-requests", owner, project)
}

// ListPullRequests는 GET .../pull-requests[?state=&author=&assignee=&label=]를 호출한다. 응답은
// PullRequest 엔티티를 직접 직렬화한 배열이라(Issue와 동일한 이유로) map으로 느슨하게 받는다.
func (c *Client) ListPullRequests(ctx context.Context, owner, project string, opts PullRequestListOptions) ([]map[string]interface{}, error) {
	path := pullRequestsBasePath(owner, project)
	q := url.Values{}
	if opts.State != "" {
		q.Set("state", opts.State)
	}
	if opts.Author != "" {
		q.Set("author", opts.Author)
	}
	if opts.Assignee != "" {
		q.Set("assignee", opts.Assignee)
	}
	if opts.Label != "" {
		q.Set("label", opts.Label)
	}
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var out []map[string]interface{}
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetPullRequest는 GET .../pull-requests/{number}를 호출한다.
func (c *Client) GetPullRequest(ctx context.Context, owner, project string, number int64) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d", pullRequestsBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreatePullRequest는 POST .../pull-requests를 호출한다.
func (c *Client) CreatePullRequest(ctx context.Context, owner, project string, req CreatePullRequestRequest) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.DoJSON(ctx, http.MethodPost, pullRequestsBasePath(owner, project), req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdatePullRequest는 PATCH .../pull-requests/{number}를 호출한다(yona-wiki P3-02 4라운드 신설
// 어댑터 — 서비스 로직 자체는 PullRequestController.updatePullRequest()가 Step5부터 이미 있었다).
func (c *Client) UpdatePullRequest(ctx context.Context, owner, project string, number int64, req UpdatePullRequestRequest) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d", pullRequestsBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodPatch, path, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// MergePullRequest는 POST .../pull-requests/{number}/merge를 호출한다.
func (c *Client) MergePullRequest(ctx context.Context, owner, project string, number int64) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d/merge", pullRequestsBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodPost, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ClosePullRequest는 POST .../pull-requests/{number}/close를 호출한다.
func (c *Client) ClosePullRequest(ctx context.Context, owner, project string, number int64) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d/close", pullRequestsBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodPost, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ReopenPullRequest는 POST .../pull-requests/{number}/reopen을 호출한다.
func (c *Client) ReopenPullRequest(ctx context.Context, owner, project string, number int64) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d/reopen", pullRequestsBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodPost, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetPullRequestDiff는 GET .../pull-requests/{number}/diff를 호출한다.
//
// yona-wiki P3-02 10라운드(TASK-0419) — 서버가 원래 FileDiff 엔티티(JGit 내부 타입 RawText/
// EditList/FileMode를 그대로 들고 있는 값 객체)를 가공 없이 그대로 반환해, pathA/pathB조차
// 신뢰할 수 없었다(실측: `pr diff`가 "- -> -"로 깨져 나옴). 서버가 이제 pathA/pathB/changeType 등
// 단순 필드와 서버가 직접 조립한 unified diff 텍스트(patch)만 담은 응답 DTO로 변환해 내려주므로
// (PullRequestController.getDiff() -> FileDiffResponse), map으로 느슨하게 받는 것 자체는 유지하되
// (다른 필드가 CLI 버전 변경 없이 늘어나도 깨지지 않도록) 이제 pathA/pathB/changeType/patch 전부
// 안정적으로 신뢰할 수 있다.
func (c *Client) GetPullRequestDiff(ctx context.Context, owner, project string, number int64) ([]map[string]interface{}, error) {
	var out []map[string]interface{}
	path := fmt.Sprintf("%s/%d/diff", pullRequestsBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// AddPullRequestComment는 POST .../pull-requests/{number}/comments를 호출한다(yona PR 전체에
// 붙는 일반 댓글 — commitId/codeRange 없이 CodeReviewService.createReviewComment()를 호출한
// 결과로 귀결된다, PullRequestApiController.addComment() 참고).
func (c *Client) AddPullRequestComment(ctx context.Context, owner, project string, number int64, req PullRequestCommentRequest) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d/comments", pullRequestsBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodPost, path, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// AddReviewer는 POST .../pull-requests/{number}/reviewers를 호출한다. yuna의 구현상 이 엔드포인트는
// "리뷰어를 지정"하는 게 아니라 "인증된 본인을 리뷰어로 등록"하는 자기등록 방식이라 별도 요청
// 본문이 없다(PullRequestController.addReviewer 참고). 응답 본문도 없다(200 OK만 반환).
func (c *Client) AddReviewer(ctx context.Context, owner, project string, number int64) error {
	path := fmt.Sprintf("%s/%d/reviewers", pullRequestsBasePath(owner, project), number)
	return c.DoJSON(ctx, http.MethodPost, path, nil, nil)
}

// RemoveReviewer는 DELETE .../pull-requests/{number}/reviewers를 호출한다(yona-wiki P3-02
// 12라운드 신설 — 서버는 removeReviewer 서비스/레거시 엔드포인트가 원래부터 있었지만 v1 REST
// API 어댑터가 없어 "리뷰어 등록은 되는데 취소는 안 되는" 갭이었다). AddReviewer와 동일하게
// 인증된 본인의 리뷰어 등록을 취소하는 자기등록 해제 방식이라 요청 본문이 없다.
func (c *Client) RemoveReviewer(ctx context.Context, owner, project string, number int64) error {
	path := fmt.Sprintf("%s/%d/reviewers", pullRequestsBasePath(owner, project), number)
	return c.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}

// setPullRequestAssigneeRequest는 web/PullRequestController.kt의 SetAssigneeRequest와 필드가
// 동일해야 한다.
type setPullRequestAssigneeRequest struct {
	UserID int64 `json:"userId"`
}

// SetPullRequestAssignee는 PUT .../pull-requests/{number}/assignee를 호출한다(yona-wiki P3-02
// 7라운드가 서버에 신설했지만 CLI 배선이 없던 갭 — 13라운드/TASK-0430이 해소).
func (c *Client) SetPullRequestAssignee(ctx context.Context, owner, project string, number, userID int64) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d/assignee", pullRequestsBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodPut, path, setPullRequestAssigneeRequest{UserID: userID}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RemovePullRequestAssignee는 DELETE .../pull-requests/{number}/assignee를 호출한다.
func (c *Client) RemovePullRequestAssignee(ctx context.Context, owner, project string, number int64) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d/assignee", pullRequestsBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodDelete, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// addPullRequestLabelRequest는 web/PullRequestController.kt의 AddPullRequestLabelRequest와
// 필드가 동일해야 한다.
type addPullRequestLabelRequest struct {
	LabelID int64 `json:"labelId"`
}

// AddPullRequestLabel은 POST .../pull-requests/{number}/labels를 호출한다(yona-wiki P3-02
// 7라운드가 서버에 신설했지만 CLI 배선이 없던 갭 — 13라운드/TASK-0430이 해소).
func (c *Client) AddPullRequestLabel(ctx context.Context, owner, project string, number, labelID int64) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d/labels", pullRequestsBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodPost, path, addPullRequestLabelRequest{LabelID: labelID}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RemovePullRequestLabel은 DELETE .../pull-requests/{number}/labels/{labelId}를 호출한다.
func (c *Client) RemovePullRequestLabel(ctx context.Context, owner, project string, number, labelID int64) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d/labels/%d", pullRequestsBasePath(owner, project), number, labelID)
	if err := c.DoJSON(ctx, http.MethodDelete, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

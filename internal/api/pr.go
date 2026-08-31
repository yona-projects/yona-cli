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

// PullRequestListOptions는 GET .../pull-requests의 선택 쿼리 파라미터를 담는다. yona PullRequest
// 엔티티엔 이슈와 달리 labels/assignee 개념이 없어(reviewers/contributor만 존재) --author만
// 지원한다(서버 web/PullRequestController.kt의 주석 그대로). Limit은 서버가 이 엔드포인트에
// 페이지네이션을 지원하지 않아(List<PullRequest> 그대로 반환) 클라이언트 사이드 슬라이싱으로
// 처리한다 — cmd 계층에서 응답을 받은 뒤 자른다.
type PullRequestListOptions struct {
	State  string
	Author string
}

func pullRequestsBasePath(owner, project string) string {
	return fmt.Sprintf("/api/v1/projects/%s/%s/pull-requests", owner, project)
}

// ListPullRequests는 GET .../pull-requests[?state=&author=]를 호출한다. 응답은 PullRequest
// 엔티티를 직접 직렬화한 배열이라(Issue와 동일한 이유로) map으로 느슨하게 받는다.
func (c *Client) ListPullRequests(ctx context.Context, owner, project string, opts PullRequestListOptions) ([]map[string]interface{}, error) {
	path := pullRequestsBasePath(owner, project)
	q := url.Values{}
	if opts.State != "" {
		q.Set("state", opts.State)
	}
	if opts.Author != "" {
		q.Set("author", opts.Author)
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

// GetPullRequestDiff는 GET .../pull-requests/{number}/diff를 호출한다. 서버 응답(List<FileDiff>)은
// JGit 내부 타입(RawText/EditList 등)을 그대로 노출하는 엔티티라 정확한 필드 구성을 신뢰할 수 없어
// (아래 참고) map으로 느슨하게 받는다.
//
// 알려진 리스크(서버 쪽 문제, 이번 CLI 작업 범위 밖 — yona-wiki 계획 문서에 보고): FileDiff.a/b는
// org.eclipse.jgit.diff.RawText, editList는 EditList(Edit 리스트)로 선언돼 있는데, 둘 다 일반
// Jackson 빈 컨벤션에 맞는 getter가 없는 JGit 내부 클래스다. 이 상태로 그대로 직렬화하면 필드가
// 거의 비거나(getter 없음) Jackson이 예외를 던질 가능성이 있다 — 즉 pathA/pathB/changeType 같은
// 단순 필드 위주로만 신뢰하고, a/b/editList/hunks 값은 방어적으로 다뤄야 한다.
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

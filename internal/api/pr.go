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

func pullRequestsBasePath(owner, project string) string {
	return fmt.Sprintf("/api/v1/projects/%s/%s/pull-requests", owner, project)
}

// ListPullRequests는 GET .../pull-requests[?state=STATE]를 호출한다. 응답은 PullRequest 엔티티를
// 직접 직렬화한 배열이라(Issue와 동일한 이유로) map으로 느슨하게 받는다.
func (c *Client) ListPullRequests(ctx context.Context, owner, project, state string) ([]map[string]interface{}, error) {
	path := pullRequestsBasePath(owner, project)
	if state != "" {
		path += "?" + url.Values{"state": {state}}.Encode()
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

// MergePullRequest는 POST .../pull-requests/{number}/merge를 호출한다.
func (c *Client) MergePullRequest(ctx context.Context, owner, project string, number int64) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d/merge", pullRequestsBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodPost, path, nil, &out); err != nil {
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

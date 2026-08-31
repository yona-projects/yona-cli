package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// CreateIssueRequest는 web/IssueController.kt의 CreateIssueRequest와 필드가 동일해야 한다
// (IssueRestApiController.create()가 그대로 역직렬화해 위임한다).
type CreateIssueRequest struct {
	Title       string  `json:"title"`
	Body        string  `json:"body,omitempty"`
	MilestoneID *int64  `json:"milestoneId,omitempty"`
	AssigneeID  *int64  `json:"assigneeId,omitempty"`
	LabelIDs    []int64 `json:"labelIds,omitempty"`
	IsDraft     bool    `json:"isDraft,omitempty"`
}

// UpdateIssueRequest는 web/IssueController.kt의 UpdateIssueRequest와 필드가 동일해야 한다.
// title/body는 그쪽 정의상 non-null이라 필수값으로 다룬다.
type UpdateIssueRequest struct {
	Title       string  `json:"title"`
	Body        string  `json:"body"`
	MilestoneID *int64  `json:"milestoneId,omitempty"`
	AssigneeID  *int64  `json:"assigneeId,omitempty"`
	LabelIDs    []int64 `json:"labelIds,omitempty"`
}

// CommentRequest는 web/CommentController.kt의 CommentRequest와 필드가 동일해야 한다.
type CommentRequest struct {
	Contents        string  `json:"contents"`
	Original        *string `json:"original,omitempty"`
	ParentCommentID *int64  `json:"parentCommentId,omitempty"`
}

// TransferIssueRequest는 web/IssueRestApiController.kt의 TransferIssueRequest와 필드가 동일해야
// 한다 — 대상 프로젝트를 owner/project 이름으로 받아 서버가 내부에서 숫자 id로 resolve한다.
type TransferIssueRequest struct {
	TargetOwner   string `json:"targetOwner"`
	TargetProject string `json:"targetProject"`
}

// IssuePage는 Spring Data Page<Issue>의 Jackson 기본 직렬화 형태 중 CLI가 실제로 쓰는
// 필드만 뽑아 담는다. Issue 자체는 JPA 엔티티를 그대로 직렬화하는 응답이라 정확한 필드 구성이
// 코드 변경에 취약하므로, 각 원소는 map[string]interface{}로 느슨하게 받는다(뷰 명령이
// number/title/state/body 등 흔한 키만 방어적으로 꺼내 쓰고, --json으로는 원문 그대로 보여준다).
type IssuePage struct {
	Content       []map[string]interface{} `json:"content"`
	TotalElements int64                    `json:"totalElements"`
	TotalPages    int64                    `json:"totalPages"`
	Number        int64                    `json:"number"`
	Size          int64                    `json:"size"`
}

// IssueListOptions는 GET .../issues의 선택 쿼리 파라미터를 담는다. yona-wiki P3-02 4라운드(Step8.5
// 서버 보강)가 IssueController.getIssues()에 assignee/label/author를 추가했고, Limit은 서버의
// Pageable(size 파라미터, IssueController.ITEMS_PER_PAGE_MAX로 상한)을 그대로 활용한다 — 서버가
// 이미 페이지네이션을 지원하므로 클라이언트 사이드 슬라이싱이 필요 없다(PR/프로젝트 목록과 다른 점).
type IssueListOptions struct {
	State    string
	Assignee string
	Label    string
	Author   string
	Limit    int
}

func issuesBasePath(owner, project string) string {
	return fmt.Sprintf("/api/v1/projects/%s/%s/issues", owner, project)
}

// ListIssues는 GET .../issues[?state=&assignee=&label=&author=&size=]를 호출한다. 값이 비어있는
// (또는 0인) 옵션은 쿼리 파라미터에서 생략한다(서버 기본 동작에 위임).
func (c *Client) ListIssues(ctx context.Context, owner, project string, opts IssueListOptions) (*IssuePage, error) {
	path := issuesBasePath(owner, project)
	q := url.Values{}
	if opts.State != "" {
		q.Set("state", opts.State)
	}
	if opts.Assignee != "" {
		q.Set("assignee", opts.Assignee)
	}
	if opts.Label != "" {
		q.Set("label", opts.Label)
	}
	if opts.Author != "" {
		q.Set("author", opts.Author)
	}
	if opts.Limit > 0 {
		q.Set("size", strconv.Itoa(opts.Limit))
	}
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var out IssuePage
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetIssue는 GET .../issues/{number}를 호출한다.
func (c *Client) GetIssue(ctx context.Context, owner, project string, number int64) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d", issuesBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateIssue는 POST .../issues를 호출한다.
func (c *Client) CreateIssue(ctx context.Context, owner, project string, req CreateIssueRequest) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.DoJSON(ctx, http.MethodPost, issuesBasePath(owner, project), req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateIssue는 PATCH .../issues/{number}를 호출한다.
func (c *Client) UpdateIssue(ctx context.Context, owner, project string, number int64, req UpdateIssueRequest) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d", issuesBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodPatch, path, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// AddIssueComment는 POST .../issues/{number}/comments를 호출한다.
func (c *Client) AddIssueComment(ctx context.Context, owner, project string, number int64, req CommentRequest) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d/comments", issuesBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodPost, path, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CloseIssue는 POST .../issues/{number}/close를 호출한다.
func (c *Client) CloseIssue(ctx context.Context, owner, project string, number int64) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d/close", issuesBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodPost, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ReopenIssue는 POST .../issues/{number}/reopen을 호출한다(yona-wiki P3-02 4라운드 신설 엔드포인트).
func (c *Client) ReopenIssue(ctx context.Context, owner, project string, number int64) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d/reopen", issuesBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodPost, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// TransferIssue는 POST .../issues/{number}/transfer를 호출한다(yona-wiki P3-02 4라운드 신설
// 엔드포인트) — 대상 프로젝트를 owner/project 이름으로 넘기면 서버가 내부에서 숫자 id로 resolve한다.
func (c *Client) TransferIssue(ctx context.Context, owner, project string, number int64, req TransferIssueRequest) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d/transfer", issuesBasePath(owner, project), number)
	if err := c.DoJSON(ctx, http.MethodPost, path, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

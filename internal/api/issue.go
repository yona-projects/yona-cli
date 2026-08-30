package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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

func issuesBasePath(owner, project string) string {
	return fmt.Sprintf("/api/v1/projects/%s/%s/issues", owner, project)
}

// ListIssues는 GET .../issues[?state=STATE]를 호출한다. state가 빈 문자열이면 쿼리 파라미터를
// 생략한다(서버 기본 동작에 위임).
func (c *Client) ListIssues(ctx context.Context, owner, project, state string) (*IssuePage, error) {
	path := issuesBasePath(owner, project)
	if state != "" {
		path += "?" + url.Values{"state": {state}}.Encode()
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

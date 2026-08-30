package api

import (
	"context"
	"fmt"
	"net/http"
)

// Project는 web/ProjectRestApiController.kt의 toProjectNode()가 만드는 응답 필드와 정확히
// 일치한다(id/owner/name/overview/vcs/scope) — 이 엔드포인트는 JPA 엔티티를 직접 직렬화하지
// 않고 컨트롤러가 직접 조립한 맵이라 필드가 안정적이다.
type Project struct {
	ID       int64  `json:"id"`
	Owner    string `json:"owner"`
	Name     string `json:"name"`
	Overview string `json:"overview"`
	VCS      string `json:"vcs"`
	Scope    string `json:"scope"`
}

// ListProjects는 GET /api/v1/projects/{owner}를 호출한다.
func (c *Client) ListProjects(ctx context.Context, owner string) ([]Project, error) {
	var out []Project
	path := fmt.Sprintf("/api/v1/projects/%s", owner)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetProject는 GET /api/v1/projects/{owner}/{project}를 호출한다.
func (c *Client) GetProject(ctx context.Context, owner, project string) (*Project, error) {
	var out Project
	path := fmt.Sprintf("/api/v1/projects/%s/%s", owner, project)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

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

// CreateProjectRequest는 web/ProjectRestApiController.kt의 CreateProjectRequest와 필드가
// 동일해야 한다. isXEnabled류 토글은 일부러 필드 자체를 두지 않았다 — Kotlin data class 기본값이
// 전부 true라, 이 CLI가 그 필드들을 생략하면(Go omitempty로 false를 보내는 대신 아예 JSON 키가
// 없으면) 서버가 기본값을 그대로 적용한다.
type CreateProjectRequest struct {
	Owner        string `json:"owner"`
	Name         string `json:"name"`
	Overview     string `json:"overview,omitempty"`
	ProjectScope string `json:"projectScope,omitempty"`
	VCS          string `json:"vcs,omitempty"`
}

// CreateProject는 POST /api/v1/projects(세그먼트 없는 bare 경로)를 호출한다. yona-wiki P3-02
// 4라운드 설계 결정대로 이 경로는 Fine-grained 스코프 토큰의 어떤 패턴과도 매칭되지 않아 세션
// 로그인/레거시 전권 토큰으로만 성공한다(GitHub Fine-grained PAT도 저장소 "생성" 자체는 지원하지
// 않는 것과 동일한 제약).
func (c *Client) CreateProject(ctx context.Context, req CreateProjectRequest) (*Project, error) {
	var out Project
	if err := c.DoJSON(ctx, http.MethodPost, "/api/v1/projects", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ForkProject는 POST /api/v1/projects/{owner}/{project}/fork를 호출한다. 응답은
// ProjectController.forkProject()가 그대로 돌려주는 JPA 엔티티라(toProjectNode를 거치지 않음)
// map으로 느슨하게 받는다.
func (c *Client) ForkProject(ctx context.Context, owner, project string) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("/api/v1/projects/%s/%s/fork", owner, project)
	if err := c.DoJSON(ctx, http.MethodPost, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateProjectRequest는 web/ProjectController.kt의 UpdateProjectRequest와 필드가 동일해야
// 한다. overview/projectScope는 그쪽 정의상 non-null 필수값이다(CLI는 read-modify-write로
// GetProject 결과를 시드값으로 채운다). 나머지 토글은 포인터로 둬 값을 명시적으로 지정할
// 때만 JSON에 실어(생략 시 서버 기본값 유지, CreateProjectRequest와 같은 이유).
type UpdateProjectRequest struct {
	Name                       *string `json:"name,omitempty"`
	Overview                   string  `json:"overview"`
	ProjectScope               string  `json:"projectScope"`
	IsCodeAccessibleMemberOnly *bool   `json:"isCodeAccessibleMemberOnly,omitempty"`
	IsUsingReviewerCount       *bool   `json:"isUsingReviewerCount,omitempty"`
	DefaultReviewerCount       *int    `json:"defaultReviewerCount,omitempty"`
	DefaultBranch              *string `json:"defaultBranch,omitempty"`
	IsCodeEnabled              *bool   `json:"isCodeEnabled,omitempty"`
	IsIssueEnabled             *bool   `json:"isIssueEnabled,omitempty"`
	IsPullRequestEnabled       *bool   `json:"isPullRequestEnabled,omitempty"`
	IsReviewEnabled            *bool   `json:"isReviewEnabled,omitempty"`
	IsMilestoneEnabled         *bool   `json:"isMilestoneEnabled,omitempty"`
	IsBoardEnabled             *bool   `json:"isBoardEnabled,omitempty"`
}

// UpdateProject는 PATCH /api/v1/projects/{owner}/{project}/settings를 호출한다("settings"
// 세그먼트를 쓰는 이유는 서버 쪽 ProjectRestApiController.kt 주석 참고 — metadata 스코프와
// 구분해 ADMINISTRATION 쓰기 권한을 강제하기 위함).
func (c *Client) UpdateProject(ctx context.Context, owner, project string, req UpdateProjectRequest) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("/api/v1/projects/%s/%s/settings", owner, project)
	if err := c.DoJSON(ctx, http.MethodPatch, path, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteProject는 DELETE /api/v1/projects/{owner}/{project}/settings를 호출한다.
func (c *Client) DeleteProject(ctx context.Context, owner, project string) error {
	path := fmt.Sprintf("/api/v1/projects/%s/%s/settings", owner, project)
	return c.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}

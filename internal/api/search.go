package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// SearchPage는 SearchRestApiController.kt가 반환하는 Page<Issue>/Page<Project>/Page<PullRequest>의
// 공통 껍데기다. 전부 JPA 엔티티를 그대로 직렬화하는 응답이라 원소는 map으로 느슨하게 받는다.
//
// yona-wiki P3-02 12라운드(2026-09-01) — "yona SearchType enum에 PULL_REQUEST 값이 없어 서버 자체에
// 대응 기능이 없다"던 이 주석은 7라운드(2026-09-01)에 이미 해소됐는데(SearchType.PULL_REQUEST 신설
// + GET /api/v1/search/prs 추가) CLI 쪽 배선만 그대로 남아있던 갭이었다 — 실측으로 발견, SearchPrs로
// 메운다.
type SearchPage struct {
	Content       []map[string]interface{} `json:"content"`
	TotalElements int64                    `json:"totalElements"`
}

// SearchIssues는 GET /api/v1/search/issues?q=...&page=&size=를 호출한다.
func (c *Client) SearchIssues(ctx context.Context, query string, page, size int) (*SearchPage, error) {
	return c.search(ctx, "/api/v1/search/issues", query, page, size)
}

// SearchProjects는 GET /api/v1/search/projects?q=...&page=&size=를 호출한다.
func (c *Client) SearchProjects(ctx context.Context, query string, page, size int) (*SearchPage, error) {
	return c.search(ctx, "/api/v1/search/projects", query, page, size)
}

// SearchPullRequests는 GET /api/v1/search/prs?q=...&page=&size=를 호출한다.
func (c *Client) SearchPullRequests(ctx context.Context, query string, page, size int) (*SearchPage, error) {
	return c.search(ctx, "/api/v1/search/prs", query, page, size)
}

func (c *Client) search(ctx context.Context, basePath, query string, page, size int) (*SearchPage, error) {
	q := url.Values{"q": {query}}
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	if size > 0 {
		q.Set("size", strconv.Itoa(size))
	}
	var out SearchPage
	if err := c.DoJSON(ctx, http.MethodGet, basePath+"?"+q.Encode(), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

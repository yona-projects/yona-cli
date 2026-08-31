package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// SearchPage는 SearchRestApiController.kt가 반환하는 Page<Issue>/Page<Project>의 공통 껍데기다.
// 둘 다 JPA 엔티티를 그대로 직렬화하는 응답이라 원소는 map으로 느슨하게 받는다.
//
// 알려진 서버 쪽 갭(yona-wiki 계획 문서에 기록됨, 이번 CLI 작업 범위 밖): yona SearchType enum에
// PULL_REQUEST 값이 없어 PR 자체를 색인하는 통합검색 기능이 서버에 없다 — 그래서 "yona search prs"는
// 이 CLI에 의도적으로 구현하지 않았다.
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

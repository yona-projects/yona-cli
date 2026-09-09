package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// yona 서버의 WikiRestApiController.kt(P3-42, /api/v1/projects/{owner}/{project}/wiki)를 그대로
// 감싼다. 위키 페이지는 DB 엔티티가 아니라 프로젝트별 bare git 저장소("<owner>/<project>.wiki.git")
// 안의 마크다운 파일이라, 이 API도 파일 경로(제목)를 그대로 리소스 식별자로 쓴다.

// CreateWikiPageRequest는 WikiRestApiController.CreateWikiPageRequest와 필드가 동일해야 한다.
type CreateWikiPageRequest struct {
	Title   string `json:"title"`
	Content string `json:"content,omitempty"`
	Message string `json:"message,omitempty"`
}

// UpdateWikiPageRequest는 WikiRestApiController.UpdateWikiPageRequest와 필드가 동일해야 한다.
// NewTitle을 채우면 이름변경(rename)까지 한 커밋으로 반영된다.
type UpdateWikiPageRequest struct {
	NewTitle string `json:"newTitle,omitempty"`
	Content  string `json:"content,omitempty"`
	Message  string `json:"message,omitempty"`
}

func wikiBasePath(owner, project string) string {
	return fmt.Sprintf("/api/v1/projects/%s/%s/wiki", owner, project)
}

// wikiTitlePath는 슬래시로 하위 경로를 표현하는 위키 페이지 제목(예: "Guides/Setup")을 URL
// 경로로 안전하게 인코딩한다 — 세그먼트별로 url.PathEscape를 적용해 슬래시 자체는 경로
// 구분자로 보존하고, 그 외 문자(공백/한글 등)만 퍼센트 인코딩한다.
func wikiTitlePath(title string) string {
	segments := strings.Split(title, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return strings.Join(segments, "/")
}

// ListWikiPages는 GET .../wiki/pages(옵션 q=검색어)를 호출한다.
func (c *Client) ListWikiPages(ctx context.Context, owner, project, query string) ([]map[string]interface{}, error) {
	path := wikiBasePath(owner, project) + "/pages"
	if query != "" {
		path += "?q=" + url.QueryEscape(query)
	}
	var out []map[string]interface{}
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWikiPage는 GET .../wiki/pages/{title}을 호출한다.
func (c *Client) GetWikiPage(ctx context.Context, owner, project, title string) (map[string]interface{}, error) {
	path := fmt.Sprintf("%s/pages/%s", wikiBasePath(owner, project), wikiTitlePath(title))
	var out map[string]interface{}
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateWikiPage는 POST .../wiki/pages를 호출한다.
func (c *Client) CreateWikiPage(ctx context.Context, owner, project string, req CreateWikiPageRequest) (map[string]interface{}, error) {
	path := wikiBasePath(owner, project) + "/pages"
	var out map[string]interface{}
	if err := c.DoJSON(ctx, http.MethodPost, path, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateWikiPage는 PUT .../wiki/pages/{title}을 호출한다(NewTitle을 채우면 이름변경까지 함께).
func (c *Client) UpdateWikiPage(ctx context.Context, owner, project, title string, req UpdateWikiPageRequest) (map[string]interface{}, error) {
	path := fmt.Sprintf("%s/pages/%s", wikiBasePath(owner, project), wikiTitlePath(title))
	var out map[string]interface{}
	if err := c.DoJSON(ctx, http.MethodPut, path, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteWikiPage는 DELETE .../wiki/pages/{title}(옵션 message=커밋 메시지)를 호출한다.
func (c *Client) DeleteWikiPage(ctx context.Context, owner, project, title, message string) error {
	path := fmt.Sprintf("%s/pages/%s", wikiBasePath(owner, project), wikiTitlePath(title))
	if message != "" {
		path += "?message=" + url.QueryEscape(message)
	}
	return c.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}

// ListWikiHistory는 GET .../wiki/history/{title}을 호출한다 — 그 페이지 파일의 리비전(커밋)
// 목록을 최신순으로 돌려준다.
func (c *Client) ListWikiHistory(ctx context.Context, owner, project, title string, pageNum, pageSize int) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("%s/history/%s", wikiBasePath(owner, project), wikiTitlePath(title))
	query := url.Values{}
	if pageNum > 0 {
		query.Set("pageNum", strconv.Itoa(pageNum))
	}
	if pageSize > 0 {
		query.Set("pageSize", strconv.Itoa(pageSize))
	}
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var out []map[string]interface{}
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetWikiDiff는 GET .../wiki/diff/{commitId}/{title}을 호출한다 — 그 리비전(커밋 vs 부모)이
// 그 페이지 파일에 반영한 unified diff 텍스트를 돌려준다.
func (c *Client) GetWikiDiff(ctx context.Context, owner, project, commitID, title string) (map[string]interface{}, error) {
	path := fmt.Sprintf("%s/diff/%s/%s", wikiBasePath(owner, project), url.PathEscape(commitID), wikiTitlePath(title))
	var out map[string]interface{}
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CompareWikiRevisions는 GET .../wiki/compare/{revA}/{revB}/{title}을 호출한다 — 임의의 두
// 리비전 사이에서 그 페이지 파일이 어떻게 달라졌는지의 unified diff 텍스트를 돌려준다.
func (c *Client) CompareWikiRevisions(ctx context.Context, owner, project, revA, revB, title string) (map[string]interface{}, error) {
	path := fmt.Sprintf("%s/compare/%s/%s/%s", wikiBasePath(owner, project), url.PathEscape(revA), url.PathEscape(revB), wikiTitlePath(title))
	var out map[string]interface{}
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

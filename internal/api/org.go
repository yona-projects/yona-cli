package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// OrganizationSummary는 OrganizationRestApiController.kt의 toOrgSummaryNode()가 만드는
// 목록 응답 필드와 정확히 일치한다.
type OrganizationSummary struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Descr string `json:"descr"`
}

// OrganizationPage는 GET /api/v1/organizations의 Page<OrganizationSummary> 응답 껍데기다.
type OrganizationPage struct {
	Content       []OrganizationSummary `json:"content"`
	TotalElements int64                 `json:"totalElements"`
}

// OrganizationProject는 OrganizationRestApiController.kt의 get()이 "projects" 배열 원소로 내려주는
// {owner, name} 쌍이다.
type OrganizationProject struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
}

// OrganizationDetail은 GET /api/v1/organizations/{name}의 응답 필드와 정확히 일치한다.
type OrganizationDetail struct {
	ID       int64                 `json:"id"`
	Name     string                `json:"name"`
	Descr    string                `json:"descr"`
	Projects []OrganizationProject `json:"projects"`
}

// ListOrganizations는 GET /api/v1/organizations[?filter=&page=]를 호출한다.
func (c *Client) ListOrganizations(ctx context.Context, filter string, page int) (*OrganizationPage, error) {
	path := "/api/v1/organizations"
	q := url.Values{}
	if filter != "" {
		q.Set("filter", filter)
	}
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var out OrganizationPage
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetOrganization은 GET /api/v1/organizations/{name}를 호출한다.
func (c *Client) GetOrganization(ctx context.Context, name string) (*OrganizationDetail, error) {
	var out OrganizationDetail
	path := fmt.Sprintf("/api/v1/organizations/%s", name)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

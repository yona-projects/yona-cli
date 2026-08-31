package api

import (
	"context"
	"fmt"
	"net/http"
)

// CreateLabelRequest는 web/LabelRestApiController.kt의 CreateLabelRequest와 필드가 동일해야 한다.
type CreateLabelRequest struct {
	Name                string `json:"name"`
	Color               string `json:"color"`
	Category            string `json:"category"`
	CategoryIsExclusive bool   `json:"categoryIsExclusive,omitempty"`
}

// UpdateLabelRequest는 web/LabelRestApiController.kt의 UpdateLabelRequest와 필드가 동일해야 한다.
type UpdateLabelRequest struct {
	Name       string `json:"name"`
	Color      string `json:"color"`
	CategoryID int64  `json:"categoryId"`
}

func labelsBasePath(owner, project string) string {
	return fmt.Sprintf("/api/v1/projects/%s/%s/labels", owner, project)
}

// ListLabels는 GET .../labels를 호출한다. 응답은 ProjectController.getProjectLabels()가
// 그대로 돌려주는 IssueLabel 엔티티 목록이라 map으로 느슨하게 받는다.
func (c *Client) ListLabels(ctx context.Context, owner, project string) ([]map[string]interface{}, error) {
	var out []map[string]interface{}
	if err := c.DoJSON(ctx, http.MethodGet, labelsBasePath(owner, project), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateLabel은 POST .../labels를 호출한다.
func (c *Client) CreateLabel(ctx context.Context, owner, project string, req CreateLabelRequest) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.DoJSON(ctx, http.MethodPost, labelsBasePath(owner, project), req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateLabel은 PATCH .../labels/{id}를 호출한다.
func (c *Client) UpdateLabel(ctx context.Context, owner, project string, id int64, req UpdateLabelRequest) (map[string]interface{}, error) {
	var out map[string]interface{}
	path := fmt.Sprintf("%s/%d", labelsBasePath(owner, project), id)
	if err := c.DoJSON(ctx, http.MethodPatch, path, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteLabel은 DELETE .../labels/{id}를 호출한다.
func (c *Client) DeleteLabel(ctx context.Context, owner, project string, id int64) error {
	path := fmt.Sprintf("%s/%d", labelsBasePath(owner, project), id)
	return c.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}

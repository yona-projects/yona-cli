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

// ResolveLabelID는 라벨 이름으로 프로젝트에 이미 존재하는 라벨의 id를 찾는다. issue/PR
// create·edit REST API는 라벨을 숫자 labelId로만 받으므로(라벨 이름을 직접 받지 않음) CLI가
// 사람이 입력하는 라벨 이름을 이 값으로 변환해야 한다(yona-wiki P3-02 13라운드/TASK-0430 —
// `issue create/edit --label`, `pr edit --add-label/--remove-label`).
func (c *Client) ResolveLabelID(ctx context.Context, owner, project, name string) (int64, error) {
	labels, err := c.ListLabels(ctx, owner, project)
	if err != nil {
		return 0, err
	}
	for _, l := range labels {
		if n, ok := l["name"].(string); ok && n == name {
			if id, ok := l["id"].(float64); ok {
				return int64(id), nil
			}
		}
	}
	return 0, fmt.Errorf("라벨 %q를 찾을 수 없습니다(프로젝트에 먼저 `yona label create`로 만들어야 합니다)", name)
}

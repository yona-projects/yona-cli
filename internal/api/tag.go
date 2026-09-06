package api

import (
	"context"
	"fmt"
	"net/http"
)

// CreateTagRequest는 web/TagRestApiController.kt의 CreateTagRequest와 필드가 동일해야 한다.
// Message를 비워두면 lightweight 태그, 채우면 annotated 태그가 만들어진다(서버 판정 그대로).
type CreateTagRequest struct {
	Name    string `json:"name"`
	Target  string `json:"target,omitempty"`
	Message string `json:"message,omitempty"`
}

func tagsBasePath(owner, project string) string {
	return fmt.Sprintf("/api/v1/projects/%s/%s/tags", owner, project)
}

// ListTags는 GET .../tags를 호출한다. 응답 필드: name, targetCommitId, annotated, message, tagger.
func (c *Client) ListTags(ctx context.Context, owner, project string) ([]map[string]interface{}, error) {
	var out []map[string]interface{}
	if err := c.DoJSON(ctx, http.MethodGet, tagsBasePath(owner, project), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateTag는 POST .../tags를 호출한다.
func (c *Client) CreateTag(ctx context.Context, owner, project string, req CreateTagRequest) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.DoJSON(ctx, http.MethodPost, tagsBasePath(owner, project), req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteTag는 DELETE .../tags/{name}를 호출한다.
func (c *Client) DeleteTag(ctx context.Context, owner, project, name string) error {
	path := fmt.Sprintf("%s/%s", tagsBasePath(owner, project), name)
	return c.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}

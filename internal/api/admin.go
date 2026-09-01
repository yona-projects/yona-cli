// admin.go는 yona-wiki P3-02 Step9 조사 결과를 반영해 시작됐지만, 웹훅/권한 "목록 조회"는
// 13라운드(TASK-0430) 기준으로 더 이상 스텁이 아니다.
//
// 조사 결과 요약(yuna 저장소 src/main/kotlin/com/github/search5/yona/web/ 기준):
//   - 백업: web/SiteApiController.kt의 GET /site/export(전체 DB JSON 백업 다운로드, 사이트
//     매니저 전용), POST /site/import(멀티파트 업로드로 전체 복원)가 실제로 존재한다. 응답이
//     JSON API로 깔끔히 설계돼 있어(export는 순수 JSON 바이트, import는 redirect) 그대로
//     연결했다.
//   - 웹훅: web/WebhookController.kt에 CRUD가 존재하지만 세션/폼 기반 레거시 MVC
//     컨트롤러다(`/projects/{owner}/{projectName}/webhooks`, `/api/v1` 네임스페이스 밖).
//     생성(POST, form-urlencoded)과 삭제(DELETE, 빈 JSON 202/200)는 구조상 CLI에서도 그대로
//     호출 가능해 연결했다. 목록 조회는 Step8.6(7라운드)이 `web/WebhookRestApiController.kt`
//     (`GET /api/v1/projects/{owner}/{project}/webhooks`)를 신설했는데, 그 사실을 몰랐던 이
//     CLI 쪽은 계속 스텁으로 남아 있었다 — 13라운드(TASK-0430)가 뒤늦게 CLI 배선을 연결했다
//     (같은 부류의 "서버엔 있는데 CLI가 못 따라간" 갭).
//   - 권한: web/ProjectMemberController.kt에 멤버 추가/역할변경/삭제(`/api/projects/{projectId}
//     (숫자 ID)/members/...`)가 JSON으로 존재해 연결했다. "현재 멤버 목록 + 역할"도 마찬가지로
//     Step8.6(7라운드)이 `web/ProjectPermissionRestApiController.kt`
//     (`GET /api/v1/projects/{owner}/{project}/permissions`)를 신설했지만 CLI가 못 따라갔던
//     것을 13라운드가 해소했다.
package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ExportBackup은 GET /site/export를 호출해 전체 데이터 백업 JSON 바이트를 그대로 반환한다.
// 사이트 매니저 권한이 필요하다(서버 쪽 checkAdmin).
func (c *Client) ExportBackup(ctx context.Context) ([]byte, error) {
	return c.DownloadRaw(ctx, "/site/export")
}

// ImportBackup은 POST /site/import에 멀티파트로 백업 파일을 업로드한다. 서버는 성공 시
// "redirect:/"(303/302)를 반환하고 실패 시 "error/400" 뷰(HTML)를 반환한다 — 어느 쪽도 JSON이
// 아니므로 호출부는 오류(4xx/5xx) 여부만으로 성공/실패를 판단해야 한다.
func (c *Client) ImportBackup(ctx context.Context, fileName string, content io.Reader) error {
	_, err := c.UploadMultipart(ctx, "/site/import", "data", fileName, content)
	return err
}

// CreateWebhook은 POST /projects/{owner}/{project}/webhooks를 form-urlencoded로 호출한다
// (web/WebhookController.kt의 파라미터 이름 그대로: payloadUrl/secret/gitPush/webhookType).
func (c *Client) CreateWebhook(ctx context.Context, owner, project, payloadURL, secret string, gitPush bool, webhookType string) error {
	values := url.Values{
		"payloadUrl":  {payloadURL},
		"webhookType": {webhookType},
		"gitPush":     {fmt.Sprintf("%t", gitPush)},
	}
	if secret != "" {
		values.Set("secret", secret)
	}
	path := fmt.Sprintf("/projects/%s/%s/webhooks", owner, project)
	_, err := c.PostForm(ctx, http.MethodPost, path, values)
	return err
}

// DeleteWebhook은 DELETE /projects/{owner}/{project}/webhooks/{id}를 호출한다.
func (c *Client) DeleteWebhook(ctx context.Context, owner, project string, id int64) error {
	path := fmt.Sprintf("/projects/%s/%s/webhooks/%d", owner, project, id)
	return c.DoJSON(ctx, http.MethodDelete, path, nil, nil)
}

// ListWebhooks는 GET /api/v1/projects/{owner}/{project}/webhooks를 호출한다(yona-wiki P3-02
// 13라운드/TASK-0430 — 서버는 7라운드부터 지원했지만 CLI 배선이 없던 갭 해소). 응답은
// WebhookController.listWebhooksJson()가 웹 화면과 동일한 노출 수준으로 그대로 돌려주는
// map 목록이라 느슨하게 받는다.
func (c *Client) ListWebhooks(ctx context.Context, owner, project string) ([]map[string]interface{}, error) {
	var out []map[string]interface{}
	path := fmt.Sprintf("/api/v1/projects/%s/%s/webhooks", owner, project)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// AddProjectMember는 POST /api/projects/{projectId}/members?loginId=...를 호출한다
// (web/ProjectMemberController.kt). projectId는 숫자 ID다 — owner/project 이름으로 먼저
// GetProject를 호출해 id를 얻어야 한다(cmd 계층에서 처리).
func (c *Client) AddProjectMember(ctx context.Context, projectID int64, loginID string) (map[string]string, error) {
	path := fmt.Sprintf("/api/projects/%d/members?%s", projectID, url.Values{"loginId": {loginID}}.Encode())
	var out map[string]string
	if err := c.DoJSON(ctx, http.MethodPost, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateProjectMemberRole은 PUT /api/projects/{projectId}/members/{userId}/role?roleId=...를 호출한다.
func (c *Client) UpdateProjectMemberRole(ctx context.Context, projectID, userID, roleID int64) (map[string]string, error) {
	path := fmt.Sprintf("/api/projects/%d/members/%d/role?%s", projectID, userID, url.Values{"roleId": {fmt.Sprintf("%d", roleID)}}.Encode())
	var out map[string]string
	if err := c.DoJSON(ctx, http.MethodPut, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// RemoveProjectMember는 DELETE /api/projects/{projectId}/members/{userId}를 호출한다.
func (c *Client) RemoveProjectMember(ctx context.Context, projectID, userID int64) (map[string]string, error) {
	path := fmt.Sprintf("/api/projects/%d/members/%d", projectID, userID)
	var out map[string]string
	if err := c.DoJSON(ctx, http.MethodDelete, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListProjectPermissions는 GET /api/v1/projects/{owner}/{project}/permissions를 호출한다
// (yona-wiki P3-02 13라운드/TASK-0430 — 서버는 7라운드부터 지원했지만 CLI 배선이 없던 갭
// 해소). owner/project 이름 기반이라 다른 admin permission 명령(숫자 projectId 기반)과 달리
// resolveProjectID를 거치지 않는다.
func (c *Client) ListProjectPermissions(ctx context.Context, owner, project string) ([]map[string]interface{}, error) {
	var out []map[string]interface{}
	path := fmt.Sprintf("/api/v1/projects/%s/%s/permissions", owner, project)
	if err := c.DoJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

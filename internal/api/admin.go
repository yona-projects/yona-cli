// admin.go는 yona-wiki P3-02 Step9 조사 결과를 그대로 반영한다.
//
// 조사 결과 요약(yuna 저장소 src/main/kotlin/com/github/search5/yona/web/ 기준):
//   - 백업: web/SiteApiController.kt의 GET /site/export(전체 DB JSON 백업 다운로드, 사이트
//     매니저 전용), POST /site/import(멀티파트 업로드로 전체 복원)가 실제로 존재한다. 응답이
//     JSON API로 깔끔히 설계돼 있어(export는 순수 JSON 바이트, import는 redirect) 그대로
//     연결했다.
//   - 웹훅: web/WebhookController.kt에 CRUD가 존재하지만 세션/폼 기반 레거시 MVC
//     컨트롤러다(`/projects/{owner}/{projectName}/webhooks`, `/api/v1` 네임스페이스 밖).
//     생성(POST, form-urlencoded)과 삭제(DELETE, 빈 JSON 202/200)는 구조상 CLI에서도 그대로
//     호출 가능해 연결했지만, 목록 조회(GET)는 Thymeleaf가 렌더링한 HTML 페이지를 반환할 뿐
//     구조화된 JSON을 전혀 주지 않아 CLI가 파싱할 수 없다 — 목록 명령은 스텁으로 남긴다.
//   - 권한: web/ProjectMemberController.kt에 멤버 추가/역할변경/삭제(`/api/projects/{projectId}
//     (숫자 ID)/members/...`)가 JSON으로 존재해 연결했다. 다만 "현재 멤버 목록 + 역할"을 그대로
//     내려주는 엔드포인트는 없다(가장 가까운 대체재인 assignableUsers는 "할당 가능한 사용자"
//     목록이지 "이미 배정된 권한 매트릭스"가 아니다) — 목록 명령은 스텁으로 남긴다.
//
// 두 "스텁" 항목(웹훅 목록/권한 목록)은 서버에 실제로 API가 없어서이지, CLI 구현 누락이
// 아니다 — yuna 쪽에 새 API를 추가하는 것은 이 CLI 프로젝트의 범위 밖이라 계획 문서에만
// 기록하고 CLI는 명확한 안내 메시지만 반환한다.
package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ErrNotSupportedByServer는 서버에 대응하는 JSON API가 없어 CLI가 구조화된 데이터를
// 돌려줄 수 없는 경우에 반환된다(위 패키지 주석 참고).
var ErrNotSupportedByServer = errors.New("yuna 서버에 이 조회를 위한 JSON API가 아직 없습니다(HTML 렌더링 전용 레거시 화면만 존재) — yona-wiki P3-02 계획 문서의 Step9 조사 결과 참고")

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

// ListWebhooks는 서버에 JSON 목록 API가 없어 항상 ErrNotSupportedByServer를 반환한다.
func (c *Client) ListWebhooks(ctx context.Context, owner, project string) error {
	return ErrNotSupportedByServer
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

// ListProjectPermissions는 서버에 "현재 멤버+역할 목록"을 내려주는 JSON API가 없어 항상
// ErrNotSupportedByServer를 반환한다.
func (c *Client) ListProjectPermissions(ctx context.Context, projectID int64) error {
	return ErrNotSupportedByServer
}

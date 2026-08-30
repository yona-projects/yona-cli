// Package api는 yuna 서버(REST API + 일부 레거시 세션 기반 컨트롤러)에 HTTP로 접근하는 얇은
// 클라이언트다. 인증은 yuna의 config/ApiTokenAuthenticationFilter.kt가 지원하는
// "Authorization: token <값>" 헤더 방식을 사용한다(대안인 "Yona-Token" 헤더도 필터가 지원하지만,
// GitHub CLI(gh)와 동일한 관례를 따르기 위해 Authorization 헤더를 기본으로 쓴다).
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client는 하나의 yuna 서버(BaseURL)에 하나의 토큰으로 접근하는 HTTP 클라이언트다.
type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// NewClient는 baseURL 끝의 슬래시를 정리하고 기본 타임아웃을 가진 Client를 만든다.
func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// APIError는 서버가 4xx/5xx를 반환했을 때의 오류다. Body에는 서버가 돌려준 원문(대개 JSON 에러
// 메시지, 경우에 따라 HTML 오류 페이지)을 그대로 담는다.
type APIError struct {
	Method     string
	Path       string
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	body := strings.TrimSpace(e.Body)
	if len(body) > 500 {
		body = body[:500] + "..."
	}
	if body == "" {
		return fmt.Sprintf("%s %s: HTTP %d", e.Method, e.Path, e.StatusCode)
	}
	return fmt.Sprintf("%s %s: HTTP %d: %s", e.Method, e.Path, e.StatusCode, body)
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("요청을 만들 수 없습니다(%s %s): %w", method, path, err)
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "token "+c.Token)
	}
	return req, nil
}

// RawRequest는 상태 코드와 무관하게 응답 바이트를 그대로 돌려준다(예: yona api 원시 호출,
// 이진 파일 다운로드/업로드, JSON이 아닌 응답을 다루는 관리자 명령). 네트워크 오류만 error로
// 반환하고, 4xx/5xx는 정상적인 *http.Response로 취급한다 — 오류 여부 판단은 호출자의 몫이다.
func (c *Client) RawRequest(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (*http.Response, []byte, error) {
	req, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return nil, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("%s %s 요청 실패: %w", method, path, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("%s %s 응답을 읽을 수 없습니다: %w", method, path, err)
	}
	return resp, data, nil
}

// DoJSON은 JSON 본문(reqBody, nil 가능)을 보내고 JSON 응답을 out에 디코딩한다(out이 nil이면
// 디코딩하지 않음). 4xx/5xx 응답은 *APIError로 반환한다.
func (c *Client) DoJSON(ctx context.Context, method, path string, reqBody interface{}, out interface{}) error {
	var reader io.Reader
	headers := map[string]string{"Accept": "application/json"}
	if reqBody != nil {
		encoded, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("요청 본문을 직렬화할 수 없습니다: %w", err)
		}
		reader = bytes.NewReader(encoded)
		headers["Content-Type"] = "application/json"
	}

	resp, data, err := c.RawRequest(ctx, method, path, reader, headers)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return &APIError{Method: method, Path: path, StatusCode: resp.StatusCode, Body: string(data)}
	}
	if out != nil && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("%s %s 응답 JSON을 해석할 수 없습니다: %w", method, path, err)
		}
	}
	return nil
}

// PostForm은 application/x-www-form-urlencoded 본문으로 POST/PUT/DELETE한다(레거시 세션 기반
// 컨트롤러, 예: 웹훅/프로젝트 멤버 관리 API가 이 형식을 쓴다). 응답 본문은 JSON이 아닐 수 있어
// (HTML 리다이렉트/에러 페이지 등) 디코딩하지 않고 상태 코드만 확인한다.
func (c *Client) PostForm(ctx context.Context, method, path string, values url.Values) ([]byte, error) {
	headers := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	resp, data, err := c.RawRequest(ctx, method, path, strings.NewReader(values.Encode()), headers)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return data, &APIError{Method: method, Path: path, StatusCode: resp.StatusCode, Body: string(data)}
	}
	return data, nil
}

// DownloadRaw는 GET 요청의 원문 바이트를 그대로 반환한다(예: 사이트 백업 export).
func (c *Client) DownloadRaw(ctx context.Context, path string) ([]byte, error) {
	resp, data, err := c.RawRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, &APIError{Method: http.MethodGet, Path: path, StatusCode: resp.StatusCode, Body: string(data)}
	}
	return data, nil
}

// UploadMultipart는 multipart/form-data로 파일 하나를 업로드한다(예: 사이트 백업 import).
func (c *Client) UploadMultipart(ctx context.Context, path, fieldName, fileName string, content io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return nil, fmt.Errorf("multipart 폼을 만들 수 없습니다: %w", err)
	}
	if _, err := io.Copy(part, content); err != nil {
		return nil, fmt.Errorf("업로드할 파일을 읽을 수 없습니다: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("multipart 폼을 마무리할 수 없습니다: %w", err)
	}

	headers := map[string]string{"Content-Type": writer.FormDataContentType()}
	resp, data, err := c.RawRequest(ctx, http.MethodPost, path, &buf, headers)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return data, &APIError{Method: http.MethodPost, Path: path, StatusCode: resp.StatusCode, Body: string(data)}
	}
	return data, nil
}

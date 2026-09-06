package sshhelper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client는 yona 서버의 /internal/ssh/** 엔드포인트 전용 HTTP 클라이언트다. 일반 PAT 인증
// (internal/api.Client)과는 완전히 다른 인증 방식(공유 시크릿 헤더)을 쓰므로 별도로 둔다.
type Client struct {
	BaseURL string
	Secret  string
	HTTP    *http.Client
}

func NewClient(cfg *Config) *Client {
	return &Client{
		BaseURL: strings.TrimRight(cfg.Server, "/"),
		Secret:  cfg.Secret,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

type AuthenticateResponse struct {
	Principal string `json:"principal"`
}

// Authenticate는 POST /internal/ssh/authenticate를 호출한다. HTTP 상태 코드를 그대로
// 반환하므로(200/403/404 각각 의미가 다름), 호출부가 상태별로 다르게 처리할 수 있다.
func (c *Client) Authenticate(ctx context.Context, publicKeyLine string) (*AuthenticateResponse, int, error) {
	reqBody, err := json.Marshal(map[string]string{"publicKey": publicKeyLine})
	if err != nil {
		return nil, 0, fmt.Errorf("요청 본문을 만들 수 없습니다: %w", err)
	}

	statusCode, respBody, err := c.post(ctx, "/internal/ssh/authenticate", reqBody)
	if err != nil {
		return nil, 0, err
	}
	if statusCode != http.StatusOK {
		return nil, statusCode, nil
	}

	var parsed AuthenticateResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, statusCode, fmt.Errorf("서버 응답을 해석할 수 없습니다: %w", err)
	}
	return &parsed, statusCode, nil
}

type AuthorizeResponse struct {
	Allowed bool   `json:"allowed"`
	RepoDir string `json:"repoDir"`
	Service string `json:"service"`
	Reason  string `json:"reason"`
}

// Authorize는 POST /internal/ssh/authorize를 호출한다.
func (c *Client) Authorize(ctx context.Context, principal, command string) (*AuthorizeResponse, error) {
	reqBody, err := json.Marshal(map[string]string{"principal": principal, "command": command})
	if err != nil {
		return nil, fmt.Errorf("요청 본문을 만들 수 없습니다: %w", err)
	}

	statusCode, respBody, err := c.post(ctx, "/internal/ssh/authorize", reqBody)
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("서버가 예기치 않은 상태코드를 반환했습니다: %d: %s", statusCode, string(respBody))
	}

	var parsed AuthorizeResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("서버 응답을 해석할 수 없습니다: %w", err)
	}
	return &parsed, nil
}

func (c *Client) post(ctx context.Context, path string, body []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, fmt.Errorf("요청을 만들 수 없습니다(%s): %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Yona-Internal-Secret", c.Secret)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("%s 요청이 실패했습니다: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("%s 응답을 읽을 수 없습니다: %w", path, err)
	}
	return resp.StatusCode, respBody, nil
}

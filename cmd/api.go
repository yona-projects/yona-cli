package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// yona-wiki P3-02 Step10 — gh api와 동일한 컨셉의 저수준 원시 HTTP 호출 명령. 아직 CLI가 감싸지
// 않은 엔드포인트를 스크립팅/디버깅 용도로 직접 두드릴 수 있게 한다.
func newAPICmd(ctx *cmdContext) *cobra.Command {
	var method string
	var fields []string
	var headers []string
	var inputFile string

	cmd := &cobra.Command{
		Use:   "api <path>",
		Short: "yona 서버 REST API를 원시 HTTP 요청으로 직접 호출한다 (디버깅/스크립팅용)",
		Long: "gh api와 동일한 컨셉이다. 예:\n" +
			`  yona api /api/v1/projects/acme` + "\n" +
			`  yona api -X POST -f title=hello -f body=world /api/v1/projects/acme/widgets/issues`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}

			client, err := ctx.newClient()
			if err != nil {
				return err
			}

			reqHeaders := map[string]string{}
			for _, h := range headers {
				k, v, ok := strings.Cut(h, ":")
				if !ok {
					return fmt.Errorf(`--header 값은 "Key: Value" 형식이어야 합니다 (입력값: %q)`, h)
				}
				reqHeaders[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}

			body, err := buildAPIRequestBody(fields, inputFile, cmd)
			if err != nil {
				return err
			}
			if body != nil {
				if _, ok := reqHeaders["Content-Type"]; !ok {
					reqHeaders["Content-Type"] = "application/json"
				}
			}

			resp, data, err := client.RawRequest(cmd.Context(), strings.ToUpper(method), path, body, reqHeaders)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			if resp.StatusCode >= 400 {
				return fmt.Errorf("HTTP %d", resp.StatusCode)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&method, "method", "X", http.MethodGet, "HTTP 메서드")
	cmd.Flags().StringArrayVarP(&fields, "field", "f", nil, `JSON 요청 본문 필드, "key=value" 형식 (여러 번 지정 가능)`)
	cmd.Flags().StringArrayVarP(&headers, "header", "H", nil, `추가 HTTP 헤더, "Key: Value" 형식 (여러 번 지정 가능)`)
	cmd.Flags().StringVar(&inputFile, "input", "", `요청 본문을 파일에서 그대로 읽는다("-"는 표준입력). --field와 함께 쓸 수 없다`)
	return cmd
}

func buildAPIRequestBody(fields []string, inputFile string, cmd *cobra.Command) (io.Reader, error) {
	if inputFile != "" && len(fields) > 0 {
		return nil, fmt.Errorf("--input과 --field는 함께 쓸 수 없습니다")
	}
	if inputFile != "" {
		if inputFile == "-" {
			data, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return nil, fmt.Errorf("표준입력을 읽을 수 없습니다: %w", err)
			}
			return bytes.NewReader(data), nil
		}
		data, err := os.ReadFile(inputFile)
		if err != nil {
			return nil, fmt.Errorf("입력 파일을 읽을 수 없습니다(%s): %w", inputFile, err)
		}
		return bytes.NewReader(data), nil
	}
	if len(fields) == 0 {
		return nil, nil
	}

	obj := map[string]string{}
	for _, f := range fields {
		k, v, ok := strings.Cut(f, "=")
		if !ok {
			return nil, fmt.Errorf(`--field 값은 "key=value" 형식이어야 합니다 (입력값: %q)`, f)
		}
		obj[k] = v
	}
	encoded, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("--field 값을 JSON으로 만들 수 없습니다: %w", err)
	}
	return bytes.NewReader(encoded), nil
}

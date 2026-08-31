package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabelList_PrintsTable(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/labels", r.URL.Path)
		_, _ = w.Write([]byte(`[{"id":1,"name":"bug"}]`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "label", "list", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "bug")
}

func TestLabelCreate_SendsRequiredFields(t *testing.T) {
	isolateConfigDir(t)
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":2,"name":"bug"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "label", "create", "--server", server.URL, "--token", "t", "--repo", "acme/widgets",
		"--name", "bug", "--color", "red", "--category", "type")

	require.NoError(t, err)
	assert.Equal(t, "bug", gotBody["name"])
	assert.Contains(t, out, "생성됨")
}

func TestLabelEdit_PatchesLabel(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/labels/2", r.URL.Path)
		assert.Equal(t, http.MethodPatch, r.Method)
		_, _ = w.Write([]byte(`{"id":2}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "label", "edit", "2", "--server", server.URL, "--token", "t", "--repo", "acme/widgets",
		"--name", "bug2", "--color", "blue", "--category-id", "1")

	require.NoError(t, err)
	assert.Contains(t, out, "수정")
}

// TASK-0421 — 실제 서버(LabelRestApiController.update)의 라벨 수정 응답 바디에는 "id" 필드가 없다
// (ProjectViewController.updateLabelForm 위임 결과가 불완전하게 내려옴). num(label, "id")로
// 서버 응답에서 id를 꺼내던 예전 구현은 이 경우 항상 "-"를 반환해 "라벨 #-을(를) 수정했습니다."로
// 깨져 나왔다 — 서버 응답 형식과 무관하게 사용자가 입력한 args[0]을 그대로 메시지에 쓰도록
// 고쳐서 이 회귀를 고정한다.
func TestLabelEdit_PrintsRequestedIdEvenWhenServerResponseOmitsId(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 실서버 재현: id 필드가 없는 불완전한 응답.
		_, _ = w.Write([]byte(`{"name":"bug2","color":"blue"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "label", "edit", "2", "--server", server.URL, "--token", "t", "--repo", "acme/widgets",
		"--name", "bug2", "--color", "blue", "--category-id", "1")

	require.NoError(t, err)
	assert.Contains(t, out, "라벨 #2을(를) 수정했습니다.")
	assert.NotContains(t, out, "#-")
}

func TestLabelDelete_DeletesLabel(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/labels/2", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	out, err := runCLI(t, "", "label", "delete", "2", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "삭제")
}

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

func TestTagList_PrintsTable(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/tags", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`[
			{"name":"v1.0","targetCommitId":"abc123","annotated":false,"message":null,"tagger":"alice"},
			{"name":"v2.0","targetCommitId":"def456","annotated":true,"message":"릴리즈 메모","tagger":"bob"}
		]`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "tag", "list", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "v1.0")
	assert.Contains(t, out, "lightweight")
	assert.Contains(t, out, "v2.0")
	assert.Contains(t, out, "annotated")
}

func TestTagList_NoTags_PrintsPlaceholder(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "tag", "list", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "태그가 없습니다")
}

func TestTagList_JSONFlag_FiltersFields(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"name":"v1.0","targetCommitId":"abc123","annotated":false}]`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "tag", "list", "--server", server.URL, "--token", "t", "--repo", "acme/widgets", "--json", "name")

	require.NoError(t, err)
	var got []map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	require.Len(t, got, 1)
	assert.Equal(t, "v1.0", got[0]["name"])
	_, hasCommit := got[0]["targetCommitId"]
	assert.False(t, hasCommit)
}

func TestTagCreate_SendsNameOnly_ForLightweightTag(t *testing.T) {
	isolateConfigDir(t)
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/tags", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"name":"v1.0","targetCommitId":"abc123","annotated":false}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "tag", "create", "v1.0", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Equal(t, "v1.0", gotBody["name"])
	_, hasMessage := gotBody["message"]
	assert.False(t, hasMessage, "message를 지정하지 않으면 요청 바디에도 없어야 한다(lightweight 태그)")
	assert.Contains(t, out, "생성됨")
	assert.Contains(t, out, "abc123")
}

func TestTagCreate_WithMessageAndTarget_SendsBoth(t *testing.T) {
	isolateConfigDir(t)
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"name":"v2.0","targetCommitId":"def456","annotated":true,"message":"릴리즈 메모"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "tag", "create", "v2.0", "--server", server.URL, "--token", "t", "--repo", "acme/widgets",
		"--target", "feature-a", "--message", "릴리즈 메모")

	require.NoError(t, err)
	assert.Equal(t, "v2.0", gotBody["name"])
	assert.Equal(t, "feature-a", gotBody["target"])
	assert.Equal(t, "릴리즈 메모", gotBody["message"])
	assert.Contains(t, out, "생성됨")
}

func TestTagDelete_DeletesTag(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/tags/v1.0", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	out, err := runCLI(t, "", "tag", "delete", "v1.0", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "삭제")
}

func TestTagCreate_RequiresNameArgument(t *testing.T) {
	isolateConfigDir(t)
	_, err := runCLI(t, "", "tag", "create", "--server", "http://example.invalid", "--token", "t", "--repo", "acme/widgets")

	require.Error(t, err)
}

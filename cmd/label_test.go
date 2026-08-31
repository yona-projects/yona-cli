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

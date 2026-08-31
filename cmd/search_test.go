package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchIssues_PrintsResults(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/search/issues", r.URL.Path)
		assert.Equal(t, "버그", r.URL.Query().Get("q"))
		_, _ = w.Write([]byte(`{"content":[{"number":1,"title":"버그 수정"}]}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "search", "issues", "버그", "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Contains(t, out, "버그 수정")
}

func TestSearchProjects_PrintsResults(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/search/projects", r.URL.Path)
		_, _ = w.Write([]byte(`{"content":[{"owner":"acme","name":"widgets"}]}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "search", "projects", "widgets", "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Contains(t, out, "acme/widgets")
}

// TestSearchPullRequests_PrintsResults — 12라운드 추가: 서버는 7라운드부터 GET /api/v1/search/prs를
// 지원했지만 CLI 배선이 없던 갭을 메운다.
func TestSearchPullRequests_PrintsResults(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/search/prs", r.URL.Path)
		assert.Equal(t, "버그", r.URL.Query().Get("q"))
		_, _ = w.Write([]byte(`{"content":[{"number":3,"title":"버그 수정 PR"}]}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "search", "prs", "버그", "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Contains(t, out, "버그 수정 PR")
}

package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPRList_PrintsTable(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests", r.URL.Path)
		_, _ = w.Write([]byte(`[{"number":1,"title":"기능 추가","state":"OPEN"}]`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "list", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "#1")
	assert.Contains(t, out, "기능 추가")
}

func TestPRView_PrintsDetail(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests/5", r.URL.Path)
		_, _ = w.Write([]byte(`{"number":5,"title":"제목","state":"OPEN","fromBranch":"feature","toBranch":"main","body":"설명"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "view", "5", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "#5 제목")
	assert.Contains(t, out, "feature -> main")
}

func TestPRCreate_SendsRequiredFields(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"number":10,"title":"새 PR"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "create", "--server", server.URL, "--token", "t", "--repo", "acme/widgets",
		"--title", "새 PR", "--from-project-id", "2", "--from-branch", "feature", "--to-branch", "main")

	require.NoError(t, err)
	assert.Contains(t, out, "#10")
}

func TestPRCreate_RequiresFromProjectID(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "pr", "create", "--server", "http://unused.invalid", "--token", "t", "--repo", "acme/widgets",
		"--title", "새 PR", "--from-branch", "feature", "--to-branch", "main")

	require.Error(t, err)
}

func TestPRMerge_ReportsConflict(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"conflicts":true}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "merge", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "충돌")
}

func TestPRMerge_ReportsSuccess(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"conflicts":false}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "merge", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "머지했습니다")
}

func TestPRReview_AddsSelfAsReviewer(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests/1/reviewers", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "review", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "리뷰어로 등록")
}

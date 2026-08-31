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

func TestIssueList_PrintsTable(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/issues", r.URL.Path)
		_, _ = w.Write([]byte(`{"content":[{"number":1,"title":"버그","state":"OPEN"}]}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "issue", "list", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "#1")
	assert.Contains(t, out, "버그")
	assert.Contains(t, out, "OPEN")
}

func TestIssueList_JSONFlagPrintsOnlyRequestedFields(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":[{"number":1,"title":"버그","body":"무시됨"}],"totalElements":1}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "issue", "list", "--server", server.URL, "--token", "t", "--repo", "acme/widgets", "--json", "number,title")

	require.NoError(t, err)
	assert.Contains(t, out, `"number": 1`)
	assert.Contains(t, out, `"title": "버그"`)
	assert.NotContains(t, out, "무시됨")
}

func TestIssueList_JSONFlagRequiresFieldList(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":[]}`))
	}))
	defer server.Close()

	_, err := runCLI(t, "", "issue", "list", "--server", server.URL, "--token", "t", "--repo", "acme/widgets", "--json=")

	require.Error(t, err)
}

func TestIssueList_RequiresRepoFlagOutsideGitContext(t *testing.T) {
	isolateConfigDir(t)
	t.Chdir(t.TempDir())

	_, err := runCLI(t, "", "issue", "list", "--server", "http://unused.invalid", "--token", "t")

	require.Error(t, err)
}

func TestIssueList_LimitFlagPassesSizeQueryParam(t *testing.T) {
	isolateConfigDir(t)
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"content":[]}`))
	}))
	defer server.Close()

	_, err := runCLI(t, "", "issue", "list", "--server", server.URL, "--token", "t", "--repo", "acme/widgets", "-L", "5")

	require.NoError(t, err)
	assert.Contains(t, gotQuery, "size=5")
}

func TestIssueView_PrintsIssueDetail(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/issues/3", r.URL.Path)
		_, _ = w.Write([]byte(`{"number":3,"title":"제목","state":"OPEN","body":"본문내용"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "issue", "view", "3", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "#3 제목")
	assert.Contains(t, out, "본문내용")
}

func TestIssueView_ErrorsOnNonNumericArg(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "issue", "view", "abc", "--server", "http://unused.invalid", "--token", "t", "--repo", "acme/widgets")

	require.Error(t, err)
}

func TestIssueCreate_SendsTitleAndBody(t *testing.T) {
	isolateConfigDir(t)
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"number":9,"title":"새 이슈"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "issue", "create", "--server", server.URL, "--token", "sekrit",
		"--repo", "acme/widgets", "--title", "새 이슈", "--body", "본문")

	require.NoError(t, err)
	assert.Equal(t, "token sekrit", gotAuth)
	assert.Contains(t, out, "#9")
}

func TestIssueCreate_RequiresTitle(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "issue", "create", "--server", "http://unused.invalid", "--token", "t", "--repo", "acme/widgets")

	require.Error(t, err)
}

func TestIssueComment_PostsComment(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/issues/1/comments", r.URL.Path)
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "issue", "comment", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets", "--body", "댓글")

	require.NoError(t, err)
	assert.Contains(t, out, "댓글을 남겼습니다")
}

func TestIssueClose_PostsClose(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/issues/1/close", r.URL.Path)
		_, _ = w.Write([]byte(`{"state":"CLOSED"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "issue", "close", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "닫았습니다")
}

func TestIssueList_ReturnsErrorOnServerForbidden(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer server.Close()

	_, err := runCLI(t, "", "issue", "list", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestIssueEdit_SendsUpdatedTitleAndPreservesBody(t *testing.T) {
	isolateConfigDir(t)
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"number":1,"title":"이전 제목","body":"기존 본문"}`))
			return
		}
		assert.Equal(t, http.MethodPatch, r.Method)
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		_, _ = w.Write([]byte(`{"number":1,"title":"새 제목"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "issue", "edit", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets", "--title", "새 제목")

	require.NoError(t, err)
	assert.Equal(t, "새 제목", gotBody["title"])
	assert.Equal(t, "기존 본문", gotBody["body"])
	assert.Contains(t, out, "수정")
}

func TestIssueEdit_RequiresTitleOrBody(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "issue", "edit", "1", "--server", "http://unused.invalid", "--token", "t", "--repo", "acme/widgets")

	require.Error(t, err)
}

func TestIssueReopen_PostsToReopenPath(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/issues/1/reopen", r.URL.Path)
		_, _ = w.Write([]byte(`{"state":"OPEN"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "issue", "reopen", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "다시 열었습니다")
}

func TestIssueTransfer_SendsTargetProject(t *testing.T) {
	isolateConfigDir(t)
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/issues/1/transfer", r.URL.Path)
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "issue", "transfer", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets", "--to", "acme/other")

	require.NoError(t, err)
	assert.Equal(t, "acme", gotBody["targetOwner"])
	assert.Equal(t, "other", gotBody["targetProject"])
	assert.Contains(t, out, "옮겼습니다")
}

func TestIssueStatus_PrintsAssignedAndCreatedCounts(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/user/issues/status", r.URL.Path)
		_, _ = w.Write([]byte(`{
			"assigned": {"openCount": 2, "closedCount": 1, "items": [{"number": 1, "title": "a"}]},
			"created": {"openCount": 0, "closedCount": 0, "items": []}
		}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "issue", "status", "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Contains(t, out, "열림 2")
	assert.Contains(t, out, "#1")
}

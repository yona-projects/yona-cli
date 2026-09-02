package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatus_PrintsAllSections(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/user/status", r.URL.Path)
		_, _ = w.Write([]byte(`{
			"assignedIssues": {"openCount": 2, "closedCount": 1, "items": [{"number": 1, "title": "담당 이슈"}]},
			"assignedPullRequests": {"openCount": 1, "closedCount": 0, "items": [{"number": 5, "title": "담당 PR"}]},
			"reviewRequests": {"openCount": 1, "closedCount": 0, "items": [{"number": 6, "title": "리뷰요청 PR"}]},
			"mentionedIssues": {"openCount": 0, "closedCount": 0, "items": []},
			"repositoryActivity": [{"id": 1, "title": "새 이슈가 등록되었습니다.", "eventType": "NEW_ISSUE", "resourceType": "ISSUE_POST", "resourceId": "10"}]
		}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "status", "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Contains(t, out, "담당 이슈")
	assert.Contains(t, out, "#1")
	assert.Contains(t, out, "담당 PR")
	assert.Contains(t, out, "리뷰요청 PR")
	assert.Contains(t, out, "(없음)")
	assert.Contains(t, out, "[NEW_ISSUE]")
	assert.Contains(t, out, "새 이슈가 등록되었습니다.")
}

func TestStatus_JSONFlagSelectsFields(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"assignedIssues": {"openCount": 3, "closedCount": 0, "items": []},
			"assignedPullRequests": {"openCount": 0, "closedCount": 0, "items": []},
			"reviewRequests": {"openCount": 0, "closedCount": 0, "items": []},
			"mentionedIssues": {"openCount": 0, "closedCount": 0, "items": []},
			"repositoryActivity": []
		}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "status", "--server", server.URL, "--token", "t", "--json", "assignedIssues")

	require.NoError(t, err)
	assert.Contains(t, out, "\"openCount\": 3")
	assert.NotContains(t, out, "reviewRequests")
}

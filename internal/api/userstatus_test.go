package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserStatus_RequestsCorrectPathAndParsesResponse(t *testing.T) {
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

	client := NewClient(server.URL, "t")
	status, err := client.GetUserStatus(context.Background())

	require.NoError(t, err)
	assert.EqualValues(t, 2, status.AssignedIssues.OpenCount)
	require.Len(t, status.AssignedIssues.Items, 1)
	assert.EqualValues(t, 1, status.AssignedPullRequests.OpenCount)
	require.Len(t, status.AssignedPullRequests.Items, 1)
	assert.EqualValues(t, 1, status.ReviewRequests.OpenCount)
	require.Len(t, status.ReviewRequests.Items, 1)
	assert.Empty(t, status.MentionedIssues.Items)
	require.Len(t, status.RepositoryActivity, 1)
	assert.Equal(t, "NEW_ISSUE", status.RepositoryActivity[0].EventType)
}

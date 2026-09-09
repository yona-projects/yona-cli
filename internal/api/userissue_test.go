package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetIssueStatus_RequestsCorrectPathAndParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/user/issues/status", r.URL.Path)
		assert.Empty(t, r.URL.RawQuery)
		_, _ = w.Write([]byte(`{
			"assigned": {"openCount": 2, "closedCount": 1, "items": [{"number": 1}], "totalElements": 3, "totalPages": 1, "page": 1},
			"created": {"openCount": 0, "closedCount": 3, "items": []},
			"commented": {"openCount": 1, "closedCount": 0, "items": []},
			"mentioned": {"openCount": 1, "closedCount": 0, "items": []},
			"favorite": {"openCount": 1, "closedCount": 0, "items": []},
			"shared": {"openCount": 1, "closedCount": 0, "items": []}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	status, err := client.GetIssueStatus(context.Background(), IssueStatusOptions{})

	require.NoError(t, err)
	assert.EqualValues(t, 2, status.Assigned.OpenCount)
	assert.EqualValues(t, 1, status.Assigned.ClosedCount)
	require.Len(t, status.Assigned.Items, 1)
	assert.EqualValues(t, 3, status.Assigned.TotalElements)
	assert.EqualValues(t, 3, status.Created.ClosedCount)
	assert.EqualValues(t, 1, status.Commented.OpenCount)
	assert.EqualValues(t, 1, status.Mentioned.OpenCount)
	assert.EqualValues(t, 1, status.Favorite.OpenCount)
	assert.EqualValues(t, 1, status.Shared.OpenCount)
}

func TestGetIssueStatus_PassesStateFilterAndPageAsQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "closed", r.URL.Query().Get("state"))
		assert.Equal(t, "bug", r.URL.Query().Get("filter"))
		assert.Equal(t, "2", r.URL.Query().Get("pageNum"))
		_, _ = w.Write([]byte(`{"assigned": {"openCount": 0, "closedCount": 0, "items": []}, "created": {"openCount": 0, "closedCount": 0, "items": []}, "commented": {"openCount": 0, "closedCount": 0, "items": []}, "mentioned": {"openCount": 0, "closedCount": 0, "items": []}, "favorite": {"openCount": 0, "closedCount": 0, "items": []}, "shared": {"openCount": 0, "closedCount": 0, "items": []}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	_, err := client.GetIssueStatus(context.Background(), IssueStatusOptions{State: "closed", Filter: "bug", Page: 2})

	require.NoError(t, err)
}

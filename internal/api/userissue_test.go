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
		_, _ = w.Write([]byte(`{
			"assigned": {"openCount": 2, "closedCount": 1, "items": [{"number": 1}]},
			"created": {"openCount": 0, "closedCount": 3, "items": []}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	status, err := client.GetIssueStatus(context.Background())

	require.NoError(t, err)
	assert.EqualValues(t, 2, status.Assigned.OpenCount)
	assert.EqualValues(t, 1, status.Assigned.ClosedCount)
	require.Len(t, status.Assigned.Items, 1)
	assert.EqualValues(t, 3, status.Created.ClosedCount)
}

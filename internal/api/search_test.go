package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchIssues_SendsQueryAndPaginationParams(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		_, _ = w.Write([]byte(`{"content":[{"number":1}],"totalElements":1}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	page, err := client.SearchIssues(context.Background(), "버그", 1, 10)

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/search/issues?page=1&q=%EB%B2%84%EA%B7%B8&size=10", gotPath)
	require.Len(t, page.Content, 1)
}

func TestSearchProjects_RequestsCorrectPath(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"content":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	_, err := client.SearchProjects(context.Background(), "widgets", 0, 0)

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/search/projects", gotPath)
}

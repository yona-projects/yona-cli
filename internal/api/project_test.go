package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProjects_ParsesResponse(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[{"id":1,"owner":"acme","name":"widgets","overview":"desc","vcs":"GIT","scope":"PUBLIC"}]`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	projects, err := client.ListProjects(context.Background(), "acme")

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/projects/acme", gotPath)
	require.Len(t, projects, 1)
	assert.Equal(t, "widgets", projects[0].Name)
	assert.Equal(t, "PUBLIC", projects[0].Scope)
}

func TestGetProject_ParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets", r.URL.Path)
		_, _ = w.Write([]byte(`{"id":1,"owner":"acme","name":"widgets","overview":"","vcs":"GIT","scope":"PUBLIC"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	project, err := client.GetProject(context.Background(), "acme", "widgets")

	require.NoError(t, err)
	assert.Equal(t, int64(1), project.ID)
	assert.Equal(t, "widgets", project.Name)
}

func TestGetProject_ReturnsAPIErrorOnNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	_, err := client.GetProject(context.Background(), "acme", "missing")

	require.Error(t, err)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
}

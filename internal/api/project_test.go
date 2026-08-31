package api

import (
	"context"
	"encoding/json"
	"io"
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

func TestCreateProject_PostsToBarePath(t *testing.T) {
	var gotPath string
	var gotBody CreateProjectRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":9,"owner":"acme","name":"newproj"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	project, err := client.CreateProject(context.Background(), CreateProjectRequest{Owner: "acme", Name: "newproj"})

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/projects", gotPath)
	assert.Equal(t, "acme", gotBody.Owner)
	assert.Equal(t, "newproj", gotBody.Name)
	assert.EqualValues(t, 9, project.ID)
}

func TestForkProject_PostsToForkPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/fork", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = w.Write([]byte(`{"owner":"bob","name":"widgets"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	forked, err := client.ForkProject(context.Background(), "acme", "widgets")

	require.NoError(t, err)
	assert.Equal(t, "bob", forked["owner"])
}

func TestUpdateProject_PatchesSettingsPath(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody UpdateProjectRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	_, err := client.UpdateProject(context.Background(), "acme", "widgets", UpdateProjectRequest{Overview: "새 설명", ProjectScope: "PUBLIC"})

	require.NoError(t, err)
	assert.Equal(t, http.MethodPatch, gotMethod)
	assert.Equal(t, "/api/v1/projects/acme/widgets/settings", gotPath)
	assert.Equal(t, "새 설명", gotBody.Overview)
}

func TestDeleteProject_DeletesSettingsPath(t *testing.T) {
	var gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	err := client.DeleteProject(context.Background(), "acme", "widgets")

	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/api/v1/projects/acme/widgets/settings", gotPath)
}

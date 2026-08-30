package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectList_PrintsTable(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme", r.URL.Path)
		_, _ = w.Write([]byte(`[{"id":1,"owner":"acme","name":"widgets","scope":"PUBLIC"}]`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "project", "list", "acme", "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Contains(t, out, "acme/widgets")
	assert.Contains(t, out, "PUBLIC")
}

func TestProjectView_PrintsDetail(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets", r.URL.Path)
		_, _ = w.Write([]byte(`{"id":7,"owner":"acme","name":"widgets","overview":"설명","vcs":"GIT","scope":"PUBLIC"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "project", "view", "acme/widgets", "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Contains(t, out, "acme/widgets (id=7)")
	assert.Contains(t, out, "설명")
}

func TestProjectView_ErrorsOnInvalidRepoFormat(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "project", "view", "no-slash", "--server", "http://unused.invalid", "--token", "t")

	require.Error(t, err)
}

func TestProjectView_ReturnsErrorOnNotFound(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := runCLI(t, "", "project", "view", "acme/missing", "--server", server.URL, "--token", "t")

	require.Error(t, err)
}

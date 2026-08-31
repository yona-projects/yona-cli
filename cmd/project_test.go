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

func TestProjectCreate_PostsToBarePath(t *testing.T) {
	isolateConfigDir(t)
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":9,"owner":"acme","name":"newproj"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "project", "create", "acme/newproj", "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/projects", gotPath)
	assert.Contains(t, out, "acme/newproj")
}

func TestProjectFork_PostsToForkPath(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/fork", r.URL.Path)
		_, _ = w.Write([]byte(`{"owner":"bob","name":"widgets"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "project", "fork", "acme/widgets", "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Contains(t, out, "bob/widgets")
}

func TestProjectEdit_SeedsCurrentOverviewAndScope(t *testing.T) {
	isolateConfigDir(t)
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"id":1,"owner":"acme","name":"widgets","overview":"기존 설명","scope":"PUBLIC"}`))
			return
		}
		assert.Equal(t, http.MethodPatch, r.Method)
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "project", "edit", "acme/widgets", "--server", server.URL, "--token", "t", "--default-branch", "develop")

	require.NoError(t, err)
	assert.Equal(t, "기존 설명", gotBody["overview"])
	assert.Equal(t, "PUBLIC", gotBody["projectScope"])
	assert.Equal(t, "develop", gotBody["defaultBranch"])
	assert.Contains(t, out, "수정")
}

func TestProjectDelete_RequiresYesFlag(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "project", "delete", "acme/widgets", "--server", "http://unused.invalid", "--token", "t")

	require.Error(t, err)
}

func TestProjectDelete_DeletesWhenYesGiven(t *testing.T) {
	isolateConfigDir(t)
	var gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "project", "delete", "acme/widgets", "--server", server.URL, "--token", "t", "--yes")

	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/api/v1/projects/acme/widgets/settings", gotPath)
	assert.Contains(t, out, "삭제")
}

func TestProjectList_LimitFlagSlicesClientSide(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"owner":"acme","name":"a"},{"owner":"acme","name":"b"}]`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "project", "list", "acme", "--server", server.URL, "--token", "t", "-L", "1")

	require.NoError(t, err)
	assert.Contains(t, out, "acme/a")
	assert.NotContains(t, out, "acme/b")
}

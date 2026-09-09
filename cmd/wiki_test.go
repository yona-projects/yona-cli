package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWikiList_PrintsTable(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/wiki/pages", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`[
			{"title":"Home","path":"Home.md","lastCommitMessage":"Create Home"},
			{"title":"Guides/Setup","path":"Guides/Setup.md","lastCommitMessage":"Create Guides/Setup"}
		]`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "wiki", "list", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "Home")
	assert.Contains(t, out, "Guides/Setup")
}

func TestWikiList_NoPages_PrintsPlaceholder(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "wiki", "list", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "위키 페이지가 없습니다")
}

func TestWikiList_QueryFlag_SendsQParam(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "guid", r.URL.Query().Get("q"))
		_, _ = w.Write([]byte(`[{"title":"Guides/Setup"}]`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "wiki", "list", "--server", server.URL, "--token", "t", "--repo", "acme/widgets", "--query", "guid")

	require.NoError(t, err)
	assert.Contains(t, out, "Guides/Setup")
}

func TestWikiView_NestedTitle_EncodesSegmentsButKeepsSlash(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/wiki/pages/Guides/Setup", r.URL.Path)
		_, _ = w.Write([]byte(`{"title":"Guides/Setup","content":"설치 안내","revision":"abc123"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "wiki", "view", "Guides/Setup", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "설치 안내")
}

func TestWikiView_JSONFlag_FiltersFields(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"title":"Home","content":"내용","revision":"abc"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "wiki", "view", "Home", "--server", server.URL, "--token", "t", "--repo", "acme/widgets", "--json", "title")

	require.NoError(t, err)
	var got map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "Home", got["title"])
	_, hasContent := got["content"]
	assert.False(t, hasContent)
}

func TestWikiCreate_SendsTitleAndContent(t *testing.T) {
	isolateConfigDir(t)
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/wiki/pages", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"title":"Home","revision":"abc123"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "wiki", "create", "Home", "--server", server.URL, "--token", "t", "--repo", "acme/widgets",
		"--content", "# Hello", "--message", "Create Home")

	require.NoError(t, err)
	assert.Equal(t, "Home", gotBody["title"])
	assert.Equal(t, "# Hello", gotBody["content"])
	assert.Equal(t, "Create Home", gotBody["message"])
	assert.Contains(t, out, "생성됨")
	assert.Contains(t, out, "abc123")
}

func TestWikiCreate_ContentFile_ReadsFromFile(t *testing.T) {
	isolateConfigDir(t)
	dir := t.TempDir()
	contentPath := filepath.Join(dir, "content.md")
	require.NoError(t, os.WriteFile(contentPath, []byte("파일에서 읽은 내용"), 0o644))

	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"title":"Home","revision":"abc"}`))
	}))
	defer server.Close()

	_, err := runCLI(t, "", "wiki", "create", "Home", "--server", server.URL, "--token", "t", "--repo", "acme/widgets",
		"--content-file", contentPath)

	require.NoError(t, err)
	assert.Equal(t, "파일에서 읽은 내용", gotBody["content"])
}

func TestWikiCreate_ContentAndContentFileTogether_Errors(t *testing.T) {
	isolateConfigDir(t)
	_, err := runCLI(t, "", "wiki", "create", "Home", "--server", "http://example.invalid", "--token", "t", "--repo", "acme/widgets",
		"--content", "x", "--content-file", "-")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "함께 쓸 수 없습니다")
}

func TestWikiEdit_WithNewTitle_SendsRenameRequest(t *testing.T) {
	isolateConfigDir(t)
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/wiki/pages/OldTitle", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		_, _ = w.Write([]byte(`{"title":"NewTitle","revision":"def456"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "wiki", "edit", "OldTitle", "--server", server.URL, "--token", "t", "--repo", "acme/widgets",
		"--new-title", "NewTitle", "--content", "본문")

	require.NoError(t, err)
	assert.Equal(t, "NewTitle", gotBody["newTitle"])
	assert.Equal(t, "본문", gotBody["content"])
	assert.Contains(t, out, "NewTitle")
	assert.Contains(t, out, "def456")
}

func TestWikiEdit_RequiresContentOrContentFile(t *testing.T) {
	isolateConfigDir(t)
	_, err := runCLI(t, "", "wiki", "edit", "Home", "--server", "http://example.invalid", "--token", "t", "--repo", "acme/widgets")

	require.Error(t, err)
}

func TestWikiDelete_DeletesPage(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/wiki/pages/Home", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	out, err := runCLI(t, "", "wiki", "delete", "Home", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "삭제")
}

func TestWikiHistory_PrintsRevisions(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/wiki/history/Home", r.URL.Path)
		_, _ = w.Write([]byte(`[
			{"id":"abc123","shortId":"abc123","shortMessage":"Update Home","authorName":"alice"},
			{"id":"def456","shortId":"def456","shortMessage":"Create Home","authorName":"alice"}
		]`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "wiki", "history", "Home", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "Update Home")
	assert.Contains(t, out, "Create Home")
}

func TestWikiDiff_PrintsPatch(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/wiki/diff/abc123/Home", r.URL.Path)
		_, _ = w.Write([]byte(`{"title":"Home","commitId":"abc123","patch":"diff --git a/Home.md b/Home.md\n+new line"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "wiki", "diff", "abc123", "Home", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "diff --git")
	assert.Contains(t, out, "+new line")
}

func TestWikiCompare_PrintsPatch(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/wiki/compare/rev1/rev2/Home", r.URL.Path)
		_, _ = w.Write([]byte(`{"title":"Home","revA":"rev1","revB":"rev2","patch":"-old\n+new"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "wiki", "compare", "rev1", "rev2", "Home", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "-old")
	assert.Contains(t, out, "+new")
}

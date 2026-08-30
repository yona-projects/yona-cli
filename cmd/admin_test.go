package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminBackupExport_WritesToStdoutByDefault(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/site/export", r.URL.Path)
		_, _ = w.Write([]byte(`{"users":[]}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "admin", "backup", "export", "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Contains(t, out, `"users":[]`)
}

func TestAdminBackupExport_WritesToFileWithOutputFlag(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"users":[]}`))
	}))
	defer server.Close()

	outputPath := filepath.Join(t.TempDir(), "backup.json")
	out, err := runCLI(t, "", "admin", "backup", "export", "--server", server.URL, "--token", "t", "--output", outputPath)

	require.NoError(t, err)
	assert.Contains(t, out, outputPath)

	data, readErr := os.ReadFile(outputPath)
	require.NoError(t, readErr)
	assert.JSONEq(t, `{"users":[]}`, string(data))
}

func TestAdminBackupExport_ErrorsWhenForbidden(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	_, err := runCLI(t, "", "admin", "backup", "export", "--server", server.URL, "--token", "t")

	require.Error(t, err)
}

func TestAdminBackupImport_UploadsFile(t *testing.T) {
	isolateConfigDir(t)
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	backupFile := filepath.Join(t.TempDir(), "backup.json")
	require.NoError(t, os.WriteFile(backupFile, []byte(`{"a":1}`), 0o600))

	out, err := runCLI(t, "", "admin", "backup", "import", backupFile, "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Equal(t, "/site/import", receivedPath)
	assert.Contains(t, out, "복원")
}

func TestAdminBackupImport_ErrorsWhenFileMissing(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "admin", "backup", "import", "/no/such/file.json", "--server", "http://unused.invalid", "--token", "t")

	require.Error(t, err)
}

func TestAdminWebhookCreate_PostsFormFields(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/projects/acme/widgets/webhooks", r.URL.Path)
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "https://example.com/hook", r.Form.Get("payloadUrl"))
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	out, err := runCLI(t, "", "admin", "webhook", "create", "--server", server.URL, "--token", "t",
		"--repo", "acme/widgets", "--url", "https://example.com/hook")

	require.NoError(t, err)
	assert.Contains(t, out, "생성했습니다")
}

func TestAdminWebhookDelete_CallsDeleteEndpoint(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/projects/acme/widgets/webhooks/9", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
	}))
	defer server.Close()

	out, err := runCLI(t, "", "admin", "webhook", "delete", "9", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "삭제했습니다")
}

func TestAdminWebhookList_ReturnsNotSupportedError(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "admin", "webhook", "list", "--server", "http://unused.invalid", "--token", "t", "--repo", "acme/widgets")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "JSON API가 아직 없습니다")
}

func TestAdminPermissionAdd_ResolvesProjectIDThenAddsMember(t *testing.T) {
	isolateConfigDir(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/acme/widgets", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":42,"owner":"acme","name":"widgets"}`))
	})
	mux.HandleFunc("/api/projects/42/members", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "alice", r.URL.Query().Get("loginId"))
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	out, err := runCLI(t, "", "admin", "permission", "add", "alice", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "alice")
}

func TestAdminPermissionUpdateRole_ResolvesProjectIDThenUpdatesRole(t *testing.T) {
	isolateConfigDir(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/acme/widgets", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":42}`))
	})
	mux.HandleFunc("/api/projects/42/members/9/role", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "2", r.URL.Query().Get("roleId"))
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	out, err := runCLI(t, "", "admin", "permission", "update-role", "9", "2", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "변경했습니다")
}

func TestAdminPermissionRemove_ResolvesProjectIDThenRemoves(t *testing.T) {
	isolateConfigDir(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/projects/acme/widgets", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":42}`))
	})
	mux.HandleFunc("/api/projects/42/members/9", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	out, err := runCLI(t, "", "admin", "permission", "remove", "9", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "제거했습니다")
}

func TestAdminPermissionList_ReturnsNotSupportedError(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "admin", "permission", "list", "--server", "http://unused.invalid", "--token", "t", "--repo", "acme/widgets")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "JSON API가 아직 없습니다")
}

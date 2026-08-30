package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportBackup_DownloadsRawJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/site/export", r.URL.Path)
		_, _ = w.Write([]byte(`{"users":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	data, err := client.ExportBackup(context.Background())

	require.NoError(t, err)
	assert.JSONEq(t, `{"users":[]}`, string(data))
}

func TestExportBackup_ReturnsAPIErrorWhenNotAdmin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	_, err := client.ExportBackup(context.Background())

	require.Error(t, err)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
}

func TestImportBackup_UploadsMultipartFile(t *testing.T) {
	var receivedContent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/site/import", r.URL.Path)
		require.NoError(t, r.ParseMultipartForm(1<<20))
		file, _, err := r.FormFile("data")
		require.NoError(t, err)
		defer file.Close()
		buf := make([]byte, 1024)
		n, _ := file.Read(buf)
		receivedContent = string(buf[:n])
		w.WriteHeader(http.StatusFound) // 302 redirect:/ 시뮬레이션
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	err := client.ImportBackup(context.Background(), "backup.json", strings.NewReader(`{"a":1}`))

	require.NoError(t, err)
	assert.Equal(t, `{"a":1}`, receivedContent)
}

func TestCreateWebhook_SendsFormEncodedFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/projects/acme/widgets/webhooks", r.URL.Path)
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "https://example.com/hook", r.Form.Get("payloadUrl"))
		assert.Equal(t, "SIMPLE", r.Form.Get("webhookType"))
		assert.Equal(t, "true", r.Form.Get("gitPush"))
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	err := client.CreateWebhook(context.Background(), "acme", "widgets", "https://example.com/hook", "", true, "SIMPLE")

	require.NoError(t, err)
}

func TestDeleteWebhook_CallsDeletePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/projects/acme/widgets/webhooks/5", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	err := client.DeleteWebhook(context.Background(), "acme", "widgets", 5)

	require.NoError(t, err)
}

func TestListWebhooks_ReturnsNotSupportedError(t *testing.T) {
	client := NewClient("http://unused.invalid", "t")
	err := client.ListWebhooks(context.Background(), "acme", "widgets")
	assert.True(t, errors.Is(err, ErrNotSupportedByServer))
}

func TestAddProjectMember_EncodesLoginIDAsQueryParam(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/projects/3/members", r.URL.Path)
		assert.Equal(t, "alice", r.URL.Query().Get("loginId"))
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	result, err := client.AddProjectMember(context.Background(), 3, "alice")

	require.NoError(t, err)
	assert.Equal(t, "success", result["status"])
}

func TestUpdateProjectMemberRole_UsesPutMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/api/projects/3/members/9/role", r.URL.Path)
		assert.Equal(t, "2", r.URL.Query().Get("roleId"))
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	_, err := client.UpdateProjectMemberRole(context.Background(), 3, 9, 2)

	require.NoError(t, err)
}

func TestRemoveProjectMember_UsesDeleteMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/projects/3/members/9", r.URL.Path)
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	_, err := client.RemoveProjectMember(context.Background(), 3, 9)

	require.NoError(t, err)
}

func TestListProjectPermissions_ReturnsNotSupportedError(t *testing.T) {
	client := NewClient("http://unused.invalid", "t")
	err := client.ListProjectPermissions(context.Background(), 1)
	assert.True(t, errors.Is(err, ErrNotSupportedByServer))
}

package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoJSON_SetsAuthorizationHeaderAndDecodesResponse(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"title":"hello"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret-token")
	var out struct {
		Title string `json:"title"`
	}
	err := client.DoJSON(context.Background(), http.MethodGet, "/x", nil, &out)

	require.NoError(t, err)
	assert.Equal(t, "token secret-token", gotAuth)
	assert.Equal(t, "hello", out.Title)
}

func TestDoJSON_MarshalsRequestBody(t *testing.T) {
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		gotBody = string(data)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	err := client.DoJSON(context.Background(), http.MethodPost, "/x", map[string]string{"title": "hi"}, nil)

	require.NoError(t, err)
	assert.JSONEq(t, `{"title":"hi"}`, gotBody)
}

func TestDoJSON_ReturnsAPIErrorOnHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"forbidden"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	err := client.DoJSON(context.Background(), http.MethodGet, "/x", nil, nil)

	require.Error(t, err)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
	assert.Contains(t, apiErr.Body, "forbidden")
}

func TestPostForm_EncodesValuesAndIgnoresNonJSONResponse(t *testing.T) {
	var gotContentType string
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		_ = r.ParseForm()
		gotForm = r.Form
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html>not json</html>`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	values := url.Values{"payloadUrl": {"https://example.com/hook"}}
	body, err := client.PostForm(context.Background(), http.MethodPost, "/x", values)

	require.NoError(t, err)
	assert.Contains(t, gotContentType, "application/x-www-form-urlencoded")
	assert.Equal(t, "https://example.com/hook", gotForm.Get("payloadUrl"))
	assert.Contains(t, string(body), "not json")
}

func TestDownloadRaw_ReturnsBytesOnSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"users":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	data, err := client.DownloadRaw(context.Background(), "/site/export")

	require.NoError(t, err)
	assert.JSONEq(t, `{"users":[]}`, string(data))
}

func TestDownloadRaw_ReturnsAPIErrorWhenForbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	_, err := client.DownloadRaw(context.Background(), "/site/export")

	require.Error(t, err)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusForbidden, apiErr.StatusCode)
}

func TestUploadMultipart_SendsFileContent(t *testing.T) {
	var gotFileContent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseMultipartForm(1<<20))
		file, _, err := r.FormFile("data")
		require.NoError(t, err)
		defer file.Close()
		data, _ := io.ReadAll(file)
		gotFileContent = string(data)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	_, err := client.UploadMultipart(context.Background(), "/site/import", "data", "backup.json", strings.NewReader(`{"a":1}`))

	require.NoError(t, err)
	assert.Equal(t, `{"a":1}`, gotFileContent)
}

func TestAPIError_ErrorMessageIncludesStatusAndBody(t *testing.T) {
	err := &APIError{Method: "GET", Path: "/x", StatusCode: 404, Body: "not found"}
	assert.Contains(t, err.Error(), "404")
	assert.Contains(t, err.Error(), "not found")
}

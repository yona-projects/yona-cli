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

func TestListIssues_AppendsStateQueryParam(t *testing.T) {
	var gotURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		_, _ = w.Write([]byte(`{"content":[{"number":1,"title":"hi"}],"totalElements":1,"totalPages":1,"number":0,"size":25}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	page, err := client.ListIssues(context.Background(), "acme", "widgets", "OPEN")

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/projects/acme/widgets/issues?state=OPEN", gotURL)
	require.Len(t, page.Content, 1)
	assert.EqualValues(t, 1, page.Content[0]["number"])
}

func TestListIssues_OmitsQueryParamWhenStateEmpty(t *testing.T) {
	var gotURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		_, _ = w.Write([]byte(`{"content":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	_, err := client.ListIssues(context.Background(), "acme", "widgets", "")

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/projects/acme/widgets/issues", gotURL)
}

func TestGetIssue_RequestsCorrectPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/issues/42", r.URL.Path)
		_, _ = w.Write([]byte(`{"number":42,"title":"bug"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	issue, err := client.GetIssue(context.Background(), "acme", "widgets", 42)

	require.NoError(t, err)
	assert.EqualValues(t, 42, issue["number"])
}

func TestCreateIssue_SendsRequestBodyAsJSON(t *testing.T) {
	var gotBody CreateIssueRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/projects/acme/widgets/issues", r.URL.Path)
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"number":1}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	_, err := client.CreateIssue(context.Background(), "acme", "widgets", CreateIssueRequest{Title: "New bug", Body: "steps"})

	require.NoError(t, err)
	assert.Equal(t, "New bug", gotBody.Title)
	assert.Equal(t, "steps", gotBody.Body)
}

func TestUpdateIssue_UsesPatchMethod(t *testing.T) {
	var gotMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	_, err := client.UpdateIssue(context.Background(), "acme", "widgets", 1, UpdateIssueRequest{Title: "t", Body: "b"})

	require.NoError(t, err)
	assert.Equal(t, http.MethodPatch, gotMethod)
}

func TestAddIssueComment_PostsToCommentsPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/issues/1/comments", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = w.Write([]byte(`{"id":9}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	comment, err := client.AddIssueComment(context.Background(), "acme", "widgets", 1, CommentRequest{Contents: "note"})

	require.NoError(t, err)
	assert.EqualValues(t, 9, comment["id"])
}

func TestCloseIssue_PostsToClosePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/issues/1/close", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = w.Write([]byte(`{"state":"CLOSED"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	issue, err := client.CloseIssue(context.Background(), "acme", "widgets", 1)

	require.NoError(t, err)
	assert.Equal(t, "CLOSED", issue["state"])
}

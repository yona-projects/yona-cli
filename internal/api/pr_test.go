package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListPullRequests_AppendsStateQueryParam(t *testing.T) {
	var gotURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		_, _ = w.Write([]byte(`[{"number":1,"title":"pr1"}]`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	prs, err := client.ListPullRequests(context.Background(), "acme", "widgets", PullRequestListOptions{State: "OPEN"})

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests?state=OPEN", gotURL)
	require.Len(t, prs, 1)
}

func TestListPullRequests_AppendsAuthorQueryParam(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	_, err := client.ListPullRequests(context.Background(), "acme", "widgets", PullRequestListOptions{Author: "alice"})

	require.NoError(t, err)
	assert.Equal(t, "alice", gotQuery.Get("author"))
}

func TestGetPullRequest_RequestsCorrectPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests/7", r.URL.Path)
		_, _ = w.Write([]byte(`{"number":7}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	pr, err := client.GetPullRequest(context.Background(), "acme", "widgets", 7)

	require.NoError(t, err)
	assert.EqualValues(t, 7, pr["number"])
}

func TestCreatePullRequest_SendsRequestBody(t *testing.T) {
	var gotBody CreatePullRequestRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		_, _ = w.Write([]byte(`{"number":1}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	_, err := client.CreatePullRequest(context.Background(), "acme", "widgets", CreatePullRequestRequest{
		Title: "feature", FromProjectID: 2, FromBranch: "feature", ToBranch: "main",
	})

	require.NoError(t, err)
	assert.Equal(t, "feature", gotBody.Title)
	assert.EqualValues(t, 2, gotBody.FromProjectID)
}

func TestMergePullRequest_PostsToMergePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests/1/merge", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = w.Write([]byte(`{"conflicts":false}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	result, err := client.MergePullRequest(context.Background(), "acme", "widgets", 1)

	require.NoError(t, err)
	assert.Equal(t, false, result["conflicts"])
}

func TestAddReviewer_PostsToReviewersPathWithNoBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests/1/reviewers", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	err := client.AddReviewer(context.Background(), "acme", "widgets", 1)

	require.NoError(t, err)
}

func TestUpdatePullRequest_UsesPatchMethodAndSendsBody(t *testing.T) {
	var gotMethod string
	var gotBody UpdatePullRequestRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		_, _ = w.Write([]byte(`{"number":1}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	body := "새 본문"
	_, err := client.UpdatePullRequest(context.Background(), "acme", "widgets", 1, UpdatePullRequestRequest{Title: "새 제목", Body: &body})

	require.NoError(t, err)
	assert.Equal(t, http.MethodPatch, gotMethod)
	assert.Equal(t, "새 제목", gotBody.Title)
	require.NotNil(t, gotBody.Body)
	assert.Equal(t, "새 본문", *gotBody.Body)
}

func TestClosePullRequest_PostsToClosePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests/1/close", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = w.Write([]byte(`{"state":"CLOSED"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	pr, err := client.ClosePullRequest(context.Background(), "acme", "widgets", 1)

	require.NoError(t, err)
	assert.Equal(t, "CLOSED", pr["state"])
}

func TestReopenPullRequest_PostsToReopenPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests/1/reopen", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = w.Write([]byte(`{"state":"OPEN"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	pr, err := client.ReopenPullRequest(context.Background(), "acme", "widgets", 1)

	require.NoError(t, err)
	assert.Equal(t, "OPEN", pr["state"])
}

func TestGetPullRequestDiff_RequestsCorrectPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests/1/diff", r.URL.Path)
		_, _ = w.Write([]byte(`[{"pathA":"a.txt","pathB":"a.txt","changeType":"MODIFY"}]`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	diffs, err := client.GetPullRequestDiff(context.Background(), "acme", "widgets", 1)

	require.NoError(t, err)
	require.Len(t, diffs, 1)
	assert.Equal(t, "a.txt", diffs[0]["pathA"])
}

func TestAddPullRequestComment_PostsBody(t *testing.T) {
	var gotBody PullRequestCommentRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests/1/comments", r.URL.Path)
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		_, _ = w.Write([]byte(`{"id":5}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	comment, err := client.AddPullRequestComment(context.Background(), "acme", "widgets", 1, PullRequestCommentRequest{Body: "댓글"})

	require.NoError(t, err)
	assert.EqualValues(t, 5, comment["id"])
	assert.Equal(t, "댓글", gotBody.Body)
}

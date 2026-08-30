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

func TestListPullRequests_AppendsStateQueryParam(t *testing.T) {
	var gotURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		_, _ = w.Write([]byte(`[{"number":1,"title":"pr1"}]`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	prs, err := client.ListPullRequests(context.Background(), "acme", "widgets", "OPEN")

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests?state=OPEN", gotURL)
	require.Len(t, prs, 1)
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

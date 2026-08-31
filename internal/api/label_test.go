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

func TestListLabels_RequestsCorrectPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/labels", r.URL.Path)
		_, _ = w.Write([]byte(`[{"id":1,"name":"bug"}]`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	labels, err := client.ListLabels(context.Background(), "acme", "widgets")

	require.NoError(t, err)
	require.Len(t, labels, 1)
	assert.Equal(t, "bug", labels[0]["name"])
}

func TestCreateLabel_PostsRequestBody(t *testing.T) {
	var gotBody CreateLabelRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		_, _ = w.Write([]byte(`{"id":2}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	_, err := client.CreateLabel(context.Background(), "acme", "widgets", CreateLabelRequest{Name: "bug", Color: "red", Category: "type"})

	require.NoError(t, err)
	assert.Equal(t, "bug", gotBody.Name)
}

func TestUpdateLabel_PatchesLabelIDPath(t *testing.T) {
	var gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"id":2}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	_, err := client.UpdateLabel(context.Background(), "acme", "widgets", 2, UpdateLabelRequest{Name: "bug2", Color: "blue", CategoryID: 1})

	require.NoError(t, err)
	assert.Equal(t, http.MethodPatch, gotMethod)
	assert.Equal(t, "/api/v1/projects/acme/widgets/labels/2", gotPath)
}

func TestDeleteLabel_DeletesLabelIDPath(t *testing.T) {
	var gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	err := client.DeleteLabel(context.Background(), "acme", "widgets", 2)

	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/api/v1/projects/acme/widgets/labels/2", gotPath)
}

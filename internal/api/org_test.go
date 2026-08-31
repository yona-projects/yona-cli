package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOrganizations_RequestsCorrectPath(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		_, _ = w.Write([]byte(`{"content":[{"id":1,"name":"acme","descr":"설명"}],"totalElements":1}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	page, err := client.ListOrganizations(context.Background(), "ac", 0)

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/organizations?filter=ac", gotPath)
	require.Len(t, page.Content, 1)
	assert.Equal(t, "acme", page.Content[0].Name)
}

func TestGetOrganization_RequestsCorrectPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/organizations/acme", r.URL.Path)
		_, _ = w.Write([]byte(`{"id":1,"name":"acme","descr":"설명","projects":[{"owner":"acme","name":"widgets"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "t")
	org, err := client.GetOrganization(context.Background(), "acme")

	require.NoError(t, err)
	assert.Equal(t, "acme", org.Name)
	require.Len(t, org.Projects, 1)
	assert.Equal(t, "widgets", org.Projects[0].Name)
}

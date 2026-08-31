package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrgList_PrintsTable(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/organizations", r.URL.Path)
		_, _ = w.Write([]byte(`{"content":[{"id":1,"name":"acme","descr":"설명"}]}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "org", "list", "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Contains(t, out, "acme")
	assert.Contains(t, out, "설명")
}

func TestOrgView_PrintsDetail(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/organizations/acme", r.URL.Path)
		_, _ = w.Write([]byte(`{"id":1,"name":"acme","descr":"설명","projects":[{"owner":"acme","name":"widgets"}]}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "org", "view", "acme", "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Contains(t, out, "acme/widgets")
}

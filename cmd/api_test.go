package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPICmd_DefaultsToGET(t *testing.T) {
	isolateConfigDir(t)
	var gotMethod, gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "api", "/api/v1/projects/acme", "--server", server.URL, "--token", "sekrit")

	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, gotMethod)
	assert.Equal(t, "token sekrit", gotAuth)
	assert.Contains(t, out, `"ok":true`)
}

func TestAPICmd_PrependsSlashToPath(t *testing.T) {
	isolateConfigDir(t)
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
	}))
	defer server.Close()

	_, err := runCLI(t, "", "api", "api/v1/projects/acme", "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Equal(t, "/api/v1/projects/acme", gotPath)
}

func TestAPICmd_MethodFlagAndFieldsBuildJSONBody(t *testing.T) {
	isolateConfigDir(t)
	var gotMethod, gotContentType, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		data, _ := io.ReadAll(r.Body)
		gotBody = string(data)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	_, err := runCLI(t, "", "api", "-X", "POST", "-f", "title=hello", "-f", "body=world",
		"/api/v1/projects/acme/widgets/issues", "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "application/json", gotContentType)
	assert.JSONEq(t, `{"title":"hello","body":"world"}`, gotBody)
}

func TestAPICmd_ReturnsErrorOnHTTPErrorStatus(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`not found`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "api", "/missing", "--server", server.URL, "--token", "t")

	require.Error(t, err)
	assert.Contains(t, out, "not found")
	assert.Contains(t, err.Error(), "404")
}

func TestAPICmd_CustomHeader(t *testing.T) {
	isolateConfigDir(t)
	var gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Custom")
	}))
	defer server.Close()

	_, err := runCLI(t, "", "api", "/x", "-H", "X-Custom: value", "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Equal(t, "value", gotHeader)
}

func TestAPICmd_RejectsInputAndFieldTogether(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "api", "/x", "--input", "-", "-f", "a=b", "--server", "http://unused.invalid", "--token", "t")

	require.Error(t, err)
}

package sshhelper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Authenticate_SendsSecretHeaderAndParsesPrincipal(t *testing.T) {
	var receivedSecret string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSecret = r.Header.Get("X-Yona-Internal-Secret")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"principal":"deploykey:1"}`))
	}))
	defer server.Close()

	client := NewClient(&Config{Server: server.URL, Secret: "shh"})
	resp, status, err := client.Authenticate(context.Background(), "ssh-ed25519 AAAA")

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "deploykey:1", resp.Principal)
	assert.Equal(t, "shh", receivedSecret)
}

func TestClient_Authenticate_ReturnsStatusCodeWithoutErrorOnNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(&Config{Server: server.URL, Secret: "shh"})
	resp, status, err := client.Authenticate(context.Background(), "ssh-ed25519 AAAA")

	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, status)
	assert.Nil(t, resp)
}

func TestClient_Authorize_ParsesAllowedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"allowed":true,"repoDir":"/data/git/o/p.git","service":"git-upload-pack"}`))
	}))
	defer server.Close()

	client := NewClient(&Config{Server: server.URL, Secret: "shh"})
	resp, err := client.Authorize(context.Background(), "sshkey:1", "git-upload-pack '/o/p.git'")

	require.NoError(t, err)
	assert.True(t, resp.Allowed)
	assert.Equal(t, "/data/git/o/p.git", resp.RepoDir)
	assert.Equal(t, "git-upload-pack", resp.Service)
}

func TestClient_Authorize_ErrorsOnServerFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(&Config{Server: server.URL, Secret: "shh"})
	_, err := client.Authorize(context.Background(), "sshkey:1", "git-upload-pack '/o/p.git'")

	require.Error(t, err)
}

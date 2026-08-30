package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withTempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(ConfigDirEnvVar, dir)
	t.Setenv(ServerEnvVar, "")
	t.Setenv(TokenEnvVar, "")
	return dir
}

func TestLoad_ReturnsEmptyConfigWhenFileMissing(t *testing.T) {
	withTempConfigDir(t)

	cfg, err := Load()

	require.NoError(t, err)
	assert.Empty(t, cfg.CurrentHost)
	assert.NotNil(t, cfg.Hosts)
	assert.Empty(t, cfg.Hosts)
}

func TestSaveAndLoad_RoundTrips(t *testing.T) {
	dir := withTempConfigDir(t)

	cfg := &Config{Hosts: map[string]Host{}}
	cfg.SetHost("https://yona.example.com", "secret-token")

	require.NoError(t, Save(cfg))

	path := filepath.Join(dir, "config.yml")
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "https://yona.example.com", loaded.CurrentHost)
	assert.Equal(t, "secret-token", loaded.Hosts["https://yona.example.com"].Token)
}

func TestRemoveHost_ClearsCurrentHostWhenMatching(t *testing.T) {
	withTempConfigDir(t)

	cfg := &Config{Hosts: map[string]Host{}}
	cfg.SetHost("https://a.example.com", "token-a")

	cfg.RemoveHost("https://a.example.com")

	assert.Empty(t, cfg.CurrentHost)
	_, ok := cfg.Hosts["https://a.example.com"]
	assert.False(t, ok)
}

func TestRemoveHost_KeepsCurrentHostWhenNotMatching(t *testing.T) {
	withTempConfigDir(t)

	cfg := &Config{Hosts: map[string]Host{}}
	cfg.SetHost("https://a.example.com", "token-a")
	cfg.Hosts["https://b.example.com"] = Host{Token: "token-b"}

	cfg.RemoveHost("https://b.example.com")

	assert.Equal(t, "https://a.example.com", cfg.CurrentHost)
}

func TestResolveServer_PrefersFlagOverEnvOverConfig(t *testing.T) {
	withTempConfigDir(t)
	cfg := &Config{Hosts: map[string]Host{}}
	cfg.SetHost("https://config.example.com", "tok")
	require.NoError(t, Save(cfg))
	t.Setenv(ServerEnvVar, "https://env.example.com")

	server, err := ResolveServer("https://flag.example.com")

	require.NoError(t, err)
	assert.Equal(t, "https://flag.example.com", server)
}

func TestResolveServer_FallsBackToEnvThenConfig(t *testing.T) {
	withTempConfigDir(t)
	cfg := &Config{Hosts: map[string]Host{}}
	cfg.SetHost("https://config.example.com", "tok")
	require.NoError(t, Save(cfg))

	server, err := ResolveServer("")
	require.NoError(t, err)
	assert.Equal(t, "https://config.example.com", server)

	t.Setenv(ServerEnvVar, "https://env.example.com")
	server, err = ResolveServer("")
	require.NoError(t, err)
	assert.Equal(t, "https://env.example.com", server)
}

func TestResolveServer_ErrorsWhenNothingConfigured(t *testing.T) {
	withTempConfigDir(t)

	_, err := ResolveServer("")

	assert.ErrorIs(t, err, ErrNoHost)
}

func TestResolveToken_PrefersFlagThenEnvThenConfig(t *testing.T) {
	withTempConfigDir(t)
	cfg := &Config{Hosts: map[string]Host{}}
	cfg.SetHost("https://yona.example.com", "config-token")
	require.NoError(t, Save(cfg))

	server, token, err := ResolveToken("https://yona.example.com", "flag-token")
	require.NoError(t, err)
	assert.Equal(t, "https://yona.example.com", server)
	assert.Equal(t, "flag-token", token)

	t.Setenv(TokenEnvVar, "env-token")
	server, token, err = ResolveToken("https://yona.example.com", "")
	require.NoError(t, err)
	assert.Equal(t, "https://yona.example.com", server)
	assert.Equal(t, "env-token", token)

	t.Setenv(TokenEnvVar, "")
	server, token, err = ResolveToken("https://yona.example.com", "")
	require.NoError(t, err)
	assert.Equal(t, "https://yona.example.com", server)
	assert.Equal(t, "config-token", token)
}

func TestResolveToken_ErrorsWhenServerHasNoStoredToken(t *testing.T) {
	withTempConfigDir(t)

	_, _, err := ResolveToken("https://unknown.example.com", "")

	assert.ErrorIs(t, err, ErrNoHost)
}

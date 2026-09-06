package sshhelper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_ReadsServerAndSecret(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ssh-helper.yml")
	require.NoError(t, os.WriteFile(path, []byte("server: http://localhost:8080\nsecret: abc123\n"), 0o600))
	t.Setenv(ConfigPathEnvVar, path)

	cfg, err := LoadConfig()

	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080", cfg.Server)
	assert.Equal(t, "abc123", cfg.Secret)
}

func TestLoadConfig_ErrorsWhenFileMissing(t *testing.T) {
	t.Setenv(ConfigPathEnvVar, filepath.Join(t.TempDir(), "does-not-exist.yml"))

	_, err := LoadConfig()

	require.Error(t, err)
}

func TestLoadConfig_ErrorsWhenSecretMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ssh-helper.yml")
	require.NoError(t, os.WriteFile(path, []byte("server: http://localhost:8080\n"), 0o600))
	t.Setenv(ConfigPathEnvVar, path)

	_, err := LoadConfig()

	require.Error(t, err)
}

func TestLoadConfig_ErrorsWhenServerMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ssh-helper.yml")
	require.NoError(t, os.WriteFile(path, []byte("secret: abc123\n"), 0o600))
	t.Setenv(ConfigPathEnvVar, path)

	_, err := LoadConfig()

	require.Error(t, err)
}

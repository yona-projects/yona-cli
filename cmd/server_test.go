package cmd

import (
	"testing"

	"github.com/yona-projects/yona-cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerList_MarksCurrentHost(t *testing.T) {
	isolateConfigDir(t)
	cfg := &config.Config{Hosts: map[string]config.Host{}}
	cfg.SetHost("https://a.example.com", "token-a")
	cfg.SetHost("https://b.example.com", "token-b")
	require.NoError(t, config.Save(cfg))

	out, err := runCLI(t, "", "server", "list")

	require.NoError(t, err)
	assert.Contains(t, out, "* https://b.example.com")
	assert.Contains(t, out, "  https://a.example.com")
}

func TestServerUse_SwitchesCurrentHostWithoutRelogin(t *testing.T) {
	isolateConfigDir(t)
	cfg := &config.Config{Hosts: map[string]config.Host{}}
	cfg.SetHost("https://a.example.com", "token-a")
	cfg.Hosts["https://b.example.com"] = config.Host{Token: "token-b"}
	require.NoError(t, config.Save(cfg))

	out, err := runCLI(t, "", "server", "use", "https://a.example.com")

	require.NoError(t, err)
	assert.Contains(t, out, "https://a.example.com")

	loaded, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "https://a.example.com", loaded.CurrentHost)
	assert.Equal(t, "token-a", loaded.Hosts["https://a.example.com"].Token)
}

func TestServerUse_ErrorsOnUnregisteredHost(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "server", "use", "https://unknown.example.com")

	assert.Error(t, err)
}

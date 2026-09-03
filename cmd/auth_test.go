package cmd

import (
	"strings"
	"testing"

	"github.com/yona-projects/yona-cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthLogin_WithTokenFlag_SavesConfig(t *testing.T) {
	isolateConfigDir(t)

	out, err := runCLI(t, "", "auth", "login", "--server", "https://yona.example.com", "--token", "secret123")

	require.NoError(t, err)
	assert.Contains(t, out, "https://yona.example.com")

	cfg, loadErr := config.Load()
	require.NoError(t, loadErr)
	assert.Equal(t, "https://yona.example.com", cfg.CurrentHost)
	assert.Equal(t, "secret123", cfg.Hosts["https://yona.example.com"].Token)
}

func TestAuthLogin_PromptsForTokenWhenFlagMissing(t *testing.T) {
	isolateConfigDir(t)

	out, err := runCLI(t, "typed-token\n", "auth", "login", "--server", "https://yona.example.com")

	require.NoError(t, err)
	assert.Contains(t, out, "Personal Access Token")

	cfg, loadErr := config.Load()
	require.NoError(t, loadErr)
	assert.Equal(t, "typed-token", cfg.Hosts["https://yona.example.com"].Token)
}

func TestAuthLogin_ErrorsWithoutServer(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "auth", "login", "--token", "x")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "서버")
}

func TestAuthLogout_RemovesStoredHost(t *testing.T) {
	isolateConfigDir(t)
	_, err := runCLI(t, "", "auth", "login", "--server", "https://yona.example.com", "--token", "t")
	require.NoError(t, err)

	out, err := runCLI(t, "", "auth", "logout", "--server", "https://yona.example.com")

	require.NoError(t, err)
	assert.Contains(t, out, "로그아웃")

	cfg, loadErr := config.Load()
	require.NoError(t, loadErr)
	_, ok := cfg.Hosts["https://yona.example.com"]
	assert.False(t, ok)
}

func TestAuthLogout_ErrorsWhenNotLoggedIn(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "auth", "logout", "--server", "https://nope.example.com")

	require.Error(t, err)
}

func TestAuthStatus_ReportsNoLoginWhenEmpty(t *testing.T) {
	isolateConfigDir(t)

	out, err := runCLI(t, "", "auth", "status")

	require.NoError(t, err)
	assert.Contains(t, out, "로그인된 yona 서버가 없습니다")
}

func TestAuthStatus_ListsHostsAndMasksToken(t *testing.T) {
	isolateConfigDir(t)
	_, err := runCLI(t, "", "auth", "login", "--server", "https://yona.example.com", "--token", "abcd1234efgh")
	require.NoError(t, err)

	out, err := runCLI(t, "", "auth", "status")

	require.NoError(t, err)
	assert.True(t, strings.Contains(out, "https://yona.example.com"))
	assert.False(t, strings.Contains(out, "abcd1234efgh"))
	assert.Contains(t, out, "abcd")
	assert.Contains(t, out, "efgh")
}

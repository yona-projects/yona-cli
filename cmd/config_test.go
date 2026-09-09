package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigSetAndGet_RoundTrips(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "config", "set", "browser", "chromium")
	require.NoError(t, err)

	out, err := runCLI(t, "", "config", "get", "browser")
	require.NoError(t, err)
	assert.Equal(t, "chromium\n", out)
}

func TestConfigGet_ErrorsOnUnknownKey(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "config", "get", "nope")

	assert.ErrorContains(t, err, "알 수 없는 설정 키")
}

func TestConfigList_ShowsKnownKeys(t *testing.T) {
	isolateConfigDir(t)

	out, err := runCLI(t, "", "config", "list")

	require.NoError(t, err)
	assert.Contains(t, out, "browser=")
}

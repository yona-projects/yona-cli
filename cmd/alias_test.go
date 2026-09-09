package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAliasSetListDelete_RoundTrips(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "alias", "set", "pv", "pr view")
	require.NoError(t, err)

	out, err := runCLI(t, "", "alias", "list")
	require.NoError(t, err)
	assert.Contains(t, out, "pv: pr view")

	_, err = runCLI(t, "", "alias", "delete", "pv")
	require.NoError(t, err)

	out, err = runCLI(t, "", "alias", "list")
	require.NoError(t, err)
	assert.NotContains(t, out, "pv")
}

func TestAliasSet_RejectsBuiltinCommandName(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "alias", "set", "issue", "pr view")

	assert.ErrorContains(t, err, "내장 명령")
}

func TestAliasDelete_ErrorsWhenNotRegistered(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "alias", "delete", "nope")

	assert.ErrorContains(t, err, "등록되지 않은 별칭")
}

func TestAlias_ExpandsToRegisteredCommand(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/issues/1", r.URL.Path)
		_, _ = w.Write([]byte(`{"number": 1, "title": "hello"}`))
	}))
	defer server.Close()

	_, err := runCLI(t, "", "alias", "set", "iv", "issue view")
	require.NoError(t, err)

	out, err := runCLI(t, "", "iv", "1", "--repo", "acme/widgets", "--server", server.URL, "--token", "t")

	require.NoError(t, err)
	assert.Contains(t, out, "hello")
}

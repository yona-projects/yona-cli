package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowse_PrintsProjectHomeURLWithNoArgs(t *testing.T) {
	isolateConfigDir(t)

	out, err := runCLI(t, "", "browse", "--server", "https://yona.example.com", "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "https://yona.example.com/acme/widgets")
}

func TestBrowse_PrintsIssueURL(t *testing.T) {
	isolateConfigDir(t)

	out, err := runCLI(t, "", "browse", "issue", "3", "--server", "https://yona.example.com", "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "https://yona.example.com/acme/widgets/issue/3")
}

func TestBrowse_PrintsPullRequestURL(t *testing.T) {
	isolateConfigDir(t)

	out, err := runCLI(t, "", "browse", "pr", "7", "--server", "https://yona.example.com", "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "https://yona.example.com/acme/widgets/pull/7")
}

func TestBrowse_ErrorsOnUnknownTarget(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "browse", "wiki", "3", "--server", "https://yona.example.com", "--token", "t", "--repo", "acme/widgets")

	assert.Error(t, err)
}

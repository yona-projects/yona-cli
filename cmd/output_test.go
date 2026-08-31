package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCmd() (*cobra.Command, *bytes.Buffer) {
	c := &cobra.Command{}
	var buf bytes.Buffer
	c.SetOut(&buf)
	return c, &buf
}

func TestPrintJSON_FiltersObjectToRequestedFields(t *testing.T) {
	cmd, buf := newTestCmd()

	err := printJSON(cmd, map[string]interface{}{"number": 1, "title": "버그", "body": "무시됨"}, "number,title")

	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"number": 1`)
	assert.Contains(t, buf.String(), `"title": "버그"`)
	assert.NotContains(t, buf.String(), "무시됨")
}

func TestPrintJSON_FiltersEachElementOfArray(t *testing.T) {
	cmd, buf := newTestCmd()

	err := printJSON(cmd, []map[string]interface{}{{"number": 1, "title": "a"}, {"number": 2, "title": "b"}}, "number")

	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"number": 1`)
	assert.Contains(t, buf.String(), `"number": 2`)
	assert.NotContains(t, buf.String(), `"title"`)
}

func TestPrintJSON_ErrorsWhenFieldsEmpty(t *testing.T) {
	cmd, _ := newTestCmd()

	err := printJSON(cmd, map[string]interface{}{"number": 1}, "")

	assert.Error(t, err)
}

func TestPrintJSON_SilentlyDropsUnknownField(t *testing.T) {
	cmd, buf := newTestCmd()

	err := printJSON(cmd, map[string]interface{}{"number": 1}, "number,nope")

	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"number": 1`)
	assert.NotContains(t, buf.String(), "nope")
}

func TestResolveRepo_UsesExplicitRepoFlagWhenGiven(t *testing.T) {
	cmd, _ := newTestCmd()

	owner, project, err := resolveRepo(cmd, "acme/widgets")

	require.NoError(t, err)
	assert.Equal(t, "acme", owner)
	assert.Equal(t, "widgets", project)
}

func TestResolveRepo_ErrorsWhenNoRepoAndNoGitContext(t *testing.T) {
	t.Chdir(t.TempDir())
	cmd, _ := newTestCmd()
	cmd.SetContext(t.Context())

	_, _, err := resolveRepo(cmd, "")

	assert.Error(t, err)
}

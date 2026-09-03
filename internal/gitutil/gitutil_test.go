package gitutil

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOwnerProject_HTTPCloneURL(t *testing.T) {
	owner, project, err := ParseOwnerProject("http://alice@yona.example.com:8080/acme/widgets.git")

	require.NoError(t, err)
	assert.Equal(t, "acme", owner)
	assert.Equal(t, "widgets", project)
}

func TestParseOwnerProject_HTTPCloneURLWithoutUserOrGitSuffix(t *testing.T) {
	owner, project, err := ParseOwnerProject("https://yona.example.com/acme/widgets")

	require.NoError(t, err)
	assert.Equal(t, "acme", owner)
	assert.Equal(t, "widgets", project)
}

func TestParseOwnerProject_ScpStyleSSHURL(t *testing.T) {
	owner, project, err := ParseOwnerProject("git@github.com:yona-projects/yona-cli.git")

	require.NoError(t, err)
	assert.Equal(t, "yona-projects", owner)
	assert.Equal(t, "yona-cli", project)
}

func TestParseOwnerProject_ErrorsOnUnparseableURL(t *testing.T) {
	_, _, err := ParseOwnerProject("not-a-url")

	assert.Error(t, err)
}

func TestParseOwnerProject_ErrorsOnEmptyURL(t *testing.T) {
	_, _, err := ParseOwnerProject("")

	assert.Error(t, err)
}

func TestDetectRepo_ReadsOriginRemoteOfCurrentDirectory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git이 설치되어 있지 않습니다")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
	}
	run("init", "-q")
	run("remote", "add", "origin", "http://yona.example.com/acme/widgets.git")

	owner, project, err := DetectRepo(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "acme", owner)
	assert.Equal(t, "widgets", project)
}

func TestDetectRepo_ErrorsOutsideGitRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git이 설치되어 있지 않습니다")
	}
	t.Chdir(t.TempDir())

	_, _, err := DetectRepo(context.Background())

	assert.Error(t, err)
}

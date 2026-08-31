package weburl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProject(t *testing.T) {
	assert.Equal(t, "https://yona.example.com/acme/widgets", Project("https://yona.example.com/", "acme", "widgets"))
}

func TestIssue(t *testing.T) {
	assert.Equal(t, "https://yona.example.com/acme/widgets/issue/3", Issue("https://yona.example.com", "acme", "widgets", 3))
}

func TestPullRequest(t *testing.T) {
	assert.Equal(t, "https://yona.example.com/acme/widgets/pull/7", PullRequest("https://yona.example.com", "acme", "widgets", 7))
}

func TestIssueList(t *testing.T) {
	assert.Equal(t, "https://yona.example.com/acme/widgets/issues", IssueList("https://yona.example.com", "acme", "widgets"))
}

func TestPullRequestList(t *testing.T) {
	assert.Equal(t, "https://yona.example.com/acme/widgets/pulls", PullRequestList("https://yona.example.com", "acme", "widgets"))
}

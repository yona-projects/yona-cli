// Package weburl은 yona 서버의 Thymeleaf 웹 UI 페이지 URL을 계산한다("--web" 플래그와
// "yona browse"가 공유한다). REST API(JSON) 경로와 달리 이 경로들은 사람이 브라우저로 보는
// 페이지라 web/ProjectViewController.kt/IssueViewController.kt/PullRequestViewController.kt의
// @GetMapping 경로를 그대로 반영한다.
package weburl

import (
	"fmt"
	"strings"
)

// Project는 "/{owner}/{projectName}" — 프로젝트 홈 화면이다.
func Project(baseURL, owner, project string) string {
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(baseURL, "/"), owner, project)
}

// Issue는 IssueViewController.kt의 "/{owner}/{projectName}/issue/{number}"다.
func Issue(baseURL, owner, project string, number int64) string {
	return fmt.Sprintf("%s/%d", Project(baseURL, owner, project)+"/issue", number)
}

// PullRequest는 PullRequestViewController.kt의 "/{owner}/{projectName}/pull/{number}"다.
func PullRequest(baseURL, owner, project string, number int64) string {
	return fmt.Sprintf("%s/%d", Project(baseURL, owner, project)+"/pull", number)
}

// IssueList는 IssueViewController.kt의 "/{owner}/{projectName}/issues"다.
func IssueList(baseURL, owner, project string) string {
	return Project(baseURL, owner, project) + "/issues"
}

// PullRequestList는 PullRequestViewController.kt의 "/{owner}/{projectName}/pulls"다.
func PullRequestList(baseURL, owner, project string) string {
	return Project(baseURL, owner, project) + "/pulls"
}

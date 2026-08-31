// Package gitutil은 로컬 git 저장소와 상호작용하는 얇은 헬퍼를 모은다 — "--repo 로컬 git
// 컨텍스트 자동감지"(gh CLI가 현재 디렉터리의 origin 리모트로 owner/repo를 알아내는 것과 동일한
// 관례)와 "yona pr checkout"(fromProject/fromBranch로 로컬 브랜치를 만드는 git 명령 시퀀스)에 쓴다.
package gitutil

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// RemoteOriginURL은 현재 작업 디렉터리에서 "git remote get-url origin"을 실행한다. git이 없거나
// 저장소 밖이거나 origin이 없으면 오류를 반환한다.
func RemoteOriginURL(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git remote origin을 확인할 수 없습니다: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ParseOwnerProject는 yuna clone URL(TemplateHelper.getCloneUrl() 참고, 형식
// "scheme://[user@]host[:port]/{owner}/{project}[.git]")과, git@host:owner/project.git 같은
// scp 스타일 SSH URL을 모두 지원해 owner/project를 뽑아낸다. gh CLI가 "github.com/owner/repo"를
// 파싱하듯, 호스트 뒤 마지막 두 경로 세그먼트를 owner/project로 취급한다.
func ParseOwnerProject(remoteURL string) (owner string, project string, err error) {
	trimmed := strings.TrimSpace(remoteURL)
	if trimmed == "" {
		return "", "", fmt.Errorf("빈 remote URL입니다")
	}

	path := trimmed
	if idx := strings.Index(trimmed, "://"); idx >= 0 {
		// scheme://[user@]host[:port]/path 형태 — scheme+host 부분을 버리고 path만 남긴다.
		rest := trimmed[idx+3:]
		if slash := strings.Index(rest, "/"); slash >= 0 {
			path = rest[slash+1:]
		} else {
			path = ""
		}
	} else if at := strings.LastIndex(trimmed, "@"); strings.Contains(trimmed, ":") && !strings.Contains(trimmed[:strings.Index(trimmed, ":")], "/") {
		// git@host:owner/project.git 같은 scp 스타일 SSH URL.
		rest := trimmed
		if at >= 0 {
			rest = trimmed[at+1:]
		}
		if colon := strings.Index(rest, ":"); colon >= 0 {
			path = rest[colon+1:]
		}
	}

	path = strings.Trim(path, "/")
	path = strings.TrimSuffix(path, ".git")
	segments := strings.Split(path, "/")
	if len(segments) < 2 || segments[len(segments)-1] == "" || segments[len(segments)-2] == "" {
		return "", "", fmt.Errorf("remote URL에서 owner/project를 알아낼 수 없습니다: %q", remoteURL)
	}
	return segments[len(segments)-2], segments[len(segments)-1], nil
}

// DetectRepo는 현재 디렉터리의 git origin remote로부터 owner/project를 자동감지한다.
// --repo/-R이 없을 때 이 함수를 호출하고, 실패하면 호출자가 "--repo 필수" 오류로 폴백해야 한다.
func DetectRepo(ctx context.Context) (owner string, project string, err error) {
	url, err := RemoteOriginURL(ctx)
	if err != nil {
		return "", "", err
	}
	return ParseOwnerProject(url)
}

// FetchBranch는 "git fetch <remoteURL> <branch>"를 현재 디렉터리에서 실행한다("yona pr checkout"의
// 1단계) — yuna의 PR은 GitHub의 refs/pull/N/head 같은 특수 ref가 아니라 실제 fromProject의 평범한
// 브랜치라 이 방식으로 충분하다.
func FetchBranch(ctx context.Context, stdout, stderr interface{ Write([]byte) (int, error) }, remoteURL, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "fetch", remoteURL, branch)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git fetch %s %s 실패: %w", remoteURL, branch, err)
	}
	return nil
}

// CheckoutNewBranchFromFetchHead는 "git checkout -B <branchName> FETCH_HEAD"를 실행한다(2단계).
// -B를 쓰는 이유: 같은 이름의 브랜치가 이미 있으면(예: 같은 PR을 두 번째로 checkout) gh pr
// checkout처럼 그 브랜치를 FETCH_HEAD로 갱신하기 위함이다(-b는 이미 존재하면 실패한다).
func CheckoutNewBranchFromFetchHead(ctx context.Context, stdout, stderr interface{ Write([]byte) (int, error) }, branchName string) error {
	cmd := exec.CommandContext(ctx, "git", "checkout", "-B", branchName, "FETCH_HEAD")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git checkout -B %s FETCH_HEAD 실패: %w", branchName, err)
	}
	return nil
}

// OpenInBrowser는 OS별 기본 브라우저로 url을 연다(--web/yona browse 공용).
func OpenInBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("브라우저를 열 수 없습니다: %w", err)
	}
	// 브라우저 프로세스가 끝날 때까지 기다리지 않는다(fire-and-forget) — 다만 좀비 프로세스가
	// 남지 않도록 백그라운드에서 종료를 거둬들인다.
	go func() { _ = cmd.Wait() }()
	return nil
}

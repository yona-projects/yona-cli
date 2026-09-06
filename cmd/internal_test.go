package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yona-projects/yona-cli/internal/sshhelper"
)

// yona-wiki P3-03 Step4 — 이 파일의 테스트는 시스템 sshd/포트 22와 전혀 무관하다(호스트 시스템을
// 건드리지 말라는 제약과는 별개 — httptest로 yona 서버의 /internal/ssh/** 계약만 흉내내고,
// git-upload-pack/git-receive-pack 실행은 이 저장소가 만든 임시 bare 저장소를 대상으로 로컬
// 프로세스를 띄우는 것뿐이다).

func writeSshHelperConfig(t *testing.T, server, secret string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "ssh-helper.yml")
	content := "server: " + server + "\nsecret: " + secret + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	t.Setenv(sshhelper.ConfigPathEnvVar, path)
}

func TestBuildAuthorizedKeysLine(t *testing.T) {
	line := BuildAuthorizedKeysLine("/usr/local/bin/yona", "sshkey:42", "ssh-ed25519", "AAAA...")

	assert.Equal(t,
		`command="/usr/local/bin/yona internal ssh-shell --principal=sshkey:42",no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty ssh-ed25519 AAAA...`,
		line,
	)
}

func TestParseSshOriginalCommand(t *testing.T) {
	assert.NoError(t, ParseSshOriginalCommand("git-upload-pack '/owner/project.git'"))
	assert.NoError(t, ParseSshOriginalCommand("git-receive-pack '/owner/project.git'"))
	assert.NoError(t, ParseSshOriginalCommand("git-upload-archive '/owner/project.git'"))
	assert.Error(t, ParseSshOriginalCommand("/bin/bash"))
	assert.Error(t, ParseSshOriginalCommand(""))
}

func TestInternalSshAuth_PrintsAuthorizedKeysLineWhenServerRecognizesKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/ssh/authenticate", r.URL.Path)
		assert.Equal(t, "test-secret", r.Header.Get("X-Yona-Internal-Secret"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"principal":"sshkey:7"}`))
	}))
	defer server.Close()
	writeSshHelperConfig(t, server.URL, "test-secret")

	out, err := runCLI(t, "", "internal", "ssh-auth", "ssh-ed25519", "AAAABBBB")

	require.NoError(t, err)
	assert.Contains(t, out, "internal ssh-shell --principal=sshkey:7")
	assert.Contains(t, out, "ssh-ed25519 AAAABBBB")
	assert.Contains(t, out, "no-pty")
}

func TestInternalSshAuth_PrintsNothingWhenKeyUnknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	writeSshHelperConfig(t, server.URL, "test-secret")

	out, err := runCLI(t, "", "internal", "ssh-auth", "ssh-ed25519", "AAAABBBB")

	require.NoError(t, err)
	assert.Empty(t, strings.TrimSpace(out))
}

func TestInternalSshAuth_ErrorsOnUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	writeSshHelperConfig(t, server.URL, "test-secret")

	_, err := runCLI(t, "", "internal", "ssh-auth", "ssh-ed25519", "AAAABBBB")

	require.Error(t, err)
}

func TestInternalSshShell_RejectsWhenNoOriginalCommand(t *testing.T) {
	writeSshHelperConfig(t, "http://127.0.0.1:0", "test-secret")
	t.Setenv("SSH_ORIGINAL_COMMAND", "")

	_, err := runCLI(t, "", "internal", "ssh-shell", "--principal=sshkey:7")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "대화형 셸")
}

func TestInternalSshShell_RejectsWhenServerDenies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/internal/ssh/authorize", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"allowed":false,"reason":"이 저장소에 접근할 권한이 없습니다."}`))
	}))
	defer server.Close()
	writeSshHelperConfig(t, server.URL, "test-secret")
	t.Setenv("SSH_ORIGINAL_COMMAND", "git-upload-pack '/gildong/private-repo.git'")

	out, err := runCLI(t, "", "internal", "ssh-shell", "--principal=sshkey:7")

	require.Error(t, err)
	assert.Contains(t, out, "접근할 권한이 없습니다")
}

// 보안 리뷰 항목 대응 — SSH_ORIGINAL_COMMAND가 git 서비스 명령이 아니면(예: 임의 셸 명령 주입
// 시도) 서버에 묻지도 않고 즉시 거부해야 한다.
func TestInternalSshShell_RejectsNonGitCommandBeforeCallingServer(t *testing.T) {
	serverCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	writeSshHelperConfig(t, server.URL, "test-secret")
	t.Setenv("SSH_ORIGINAL_COMMAND", "/bin/bash -c 'rm -rf /'")

	_, err := runCLI(t, "", "internal", "ssh-shell", "--principal=sshkey:7")

	require.Error(t, err)
	assert.False(t, serverCalled, "지원하지 않는 명령이면 서버를 호출하지 않아야 한다")
}

// 실제 git 바이너리를 대상 디렉터리(이 테스트가 만든 임시 bare 저장소)에 대고 호출하는 통합
// 테스트 — 작업 지시문이 명시한 "실제 git 프로세스 exec 부분은 실제 git 바이너리로 검증"에 대응.
// 시스템 sshd와는 무관한, 이 프로세스 내부의 로컬 git 프로세스 실행이다.
func TestInternalSshShell_ExecsRealGitUploadPackAgainstBareRepo(t *testing.T) {
	if _, err := exec.LookPath("git-upload-pack"); err != nil {
		t.Skip("git-upload-pack 바이너리를 찾을 수 없어 스킵합니다")
	}

	repoDir := filepath.Join(t.TempDir(), "repo.git")
	initCmd := exec.Command("git", "init", "--bare", repoDir)
	require.NoError(t, initCmd.Run())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"allowed":true,"repoDir":"` + repoDir + `","service":"git-upload-pack"}`))
	}))
	defer server.Close()
	writeSshHelperConfig(t, server.URL, "test-secret")
	t.Setenv("SSH_ORIGINAL_COMMAND", "git-upload-pack '/gildong/repo.git'")

	// stdin을 즉시 닫아 git-upload-pack이 ref 광고를 쓴 뒤 종료하게 한다(실제 클라이언트는
	// 협상을 계속하지만, 이 테스트는 "올바른 저장소를 대상으로 실제 git 프로세스가 실행됐는지"만
	// 확인하면 충분하다).
	out, _ := runCLI(t, "", "internal", "ssh-shell", "--principal=sshkey:7")

	// git-upload-pack의 pkt-line 응답은 "papabilities^{}" 캡션이 포함된 참조 광고로 시작한다.
	assert.Contains(t, out, "capabilities")
}

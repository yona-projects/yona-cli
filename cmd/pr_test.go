package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPRList_PrintsTable(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests", r.URL.Path)
		_, _ = w.Write([]byte(`[{"number":1,"title":"기능 추가","state":"OPEN"}]`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "list", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "#1")
	assert.Contains(t, out, "기능 추가")
}

func TestPRView_PrintsDetail(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests/5", r.URL.Path)
		_, _ = w.Write([]byte(`{"number":5,"title":"제목","state":"OPEN","fromBranch":"feature","toBranch":"main","body":"설명"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "view", "5", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "#5 제목")
	assert.Contains(t, out, "feature -> main")
}

func TestPRCreate_ResolvesFromRepoToProjectID(t *testing.T) {
	isolateConfigDir(t)
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects/forker/widgets":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":2,"owner":"forker","name":"widgets"}`))
		case r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"number":10,"title":"새 PR"}`))
		default:
			t.Fatalf("예상치 못한 요청: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "create", "--server", server.URL, "--token", "t", "--repo", "acme/widgets",
		"--title", "새 PR", "--from", "forker/widgets", "--from-branch", "feature", "--to-branch", "main")

	require.NoError(t, err)
	assert.Contains(t, out, "#10")
	assert.EqualValues(t, 2, gotBody["fromProjectId"])
}

func TestPRCreate_RequiresFrom(t *testing.T) {
	isolateConfigDir(t)

	_, err := runCLI(t, "", "pr", "create", "--server", "http://unused.invalid", "--token", "t", "--repo", "acme/widgets",
		"--title", "새 PR", "--from-branch", "feature", "--to-branch", "main")

	require.Error(t, err)
}

func TestPRMerge_ReportsConflict(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"conflicts":true}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "merge", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "충돌")
}

func TestPRMerge_ReportsSuccess(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"conflicts":false}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "merge", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "머지했습니다")
}

func TestPRReview_AddsSelfAsReviewer(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests/1/reviewers", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "review", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "리뷰어로 등록")
}

func TestPREdit_SendsUpdatedTitleAndBody(t *testing.T) {
	isolateConfigDir(t)
	var gotMethod string
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		_, _ = w.Write([]byte(`{"number":1,"title":"새 제목"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "edit", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets", "--title", "새 제목", "--body", "새 본문")

	require.NoError(t, err)
	assert.Equal(t, http.MethodPatch, gotMethod)
	assert.Equal(t, "새 제목", gotBody["title"])
	assert.Equal(t, "새 본문", gotBody["body"])
	assert.Contains(t, out, "수정")
}

func TestPREdit_FetchesCurrentTitleWhenOmitted(t *testing.T) {
	isolateConfigDir(t)
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"number":1,"title":"기존 제목"}`))
			return
		}
		data, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(data, &gotBody))
		_, _ = w.Write([]byte(`{"number":1}`))
	}))
	defer server.Close()

	_, err := runCLI(t, "", "pr", "edit", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets", "--body", "새 본문")

	require.NoError(t, err)
	assert.Equal(t, "기존 제목", gotBody["title"])
}

func TestPRClose_PostsToClosePath(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests/1/close", r.URL.Path)
		_, _ = w.Write([]byte(`{"state":"CLOSED"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "close", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "닫았습니다")
}

func TestPRReopen_PostsToReopenPath(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests/1/reopen", r.URL.Path)
		_, _ = w.Write([]byte(`{"state":"OPEN"}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "reopen", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "다시 열었습니다")
}

func TestPRDiff_PrintsChangedFiles(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests/1/diff", r.URL.Path)
		_, _ = w.Write([]byte(`[{"pathA":"a.txt","pathB":"a.txt","changeType":"MODIFY"}]`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "diff", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "a.txt")
	assert.Contains(t, out, "MODIFY")
}

// TASK-0419 — 서버가 FileDiffResponse로 pathA/pathB/changeType과 함께 unified diff 텍스트(patch)를
// 내려주기 시작한 뒤, CLI도 그 patch 내용을 실제로 화면에 보여줘야 한다("diff" 커맨드인데 파일 목록
// 요약 한 줄만 보여주고 실제 diff 내용을 --json 없이는 볼 수 없던 UX 결함 수정).
func TestPRDiff_PrintsPatchContent(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"pathA":"a.txt","pathB":"a.txt","changeType":"MODIFY","patch":"@@ -1,1 +1,1 @@\n-old\n+new\n"}]`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "diff", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.Contains(t, out, "@@ -1,1 +1,1 @@")
	assert.Contains(t, out, "-old")
	assert.Contains(t, out, "+new")
}

// patch 필드가 없는(예: 서버 구버전) 경우 str()이 반환하는 "-" 자리표시자를 실제 diff 내용처럼
// 출력하면 안 된다.
func TestPRDiff_OmitsPlaceholderWhenPatchMissing(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"pathA":"a.txt","pathB":"a.txt","changeType":"MODIFY"}]`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "diff", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets")

	require.NoError(t, err)
	assert.NotContains(t, out, "MODIFY\ta.txt -> a.txt\n-\n")
}

func TestPRComment_PostsComment(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/projects/acme/widgets/pull-requests/1/comments", r.URL.Path)
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "comment", "1", "--server", server.URL, "--token", "t", "--repo", "acme/widgets", "--body", "댓글")

	require.NoError(t, err)
	assert.Contains(t, out, "댓글을 남겼습니다")
}

func TestPRList_LimitFlagSlicesClientSide(t *testing.T) {
	isolateConfigDir(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"number":1,"title":"a"},{"number":2,"title":"b"},{"number":3,"title":"c"}]`))
	}))
	defer server.Close()

	out, err := runCLI(t, "", "pr", "list", "--server", server.URL, "--token", "t", "--repo", "acme/widgets", "-L", "2")

	require.NoError(t, err)
	assert.Contains(t, out, "#1")
	assert.Contains(t, out, "#2")
	assert.NotContains(t, out, "#3")
}

func TestPlanCheckout_ComputesRemoteURLBranchAndLocalBranch(t *testing.T) {
	pr := map[string]interface{}{
		"fromBranch": "feature",
		"fromProject": map[string]interface{}{
			"owner": "bob",
			"name":  "widgets",
		},
	}

	remoteURL, branch, localBranch, err := planCheckout(pr, "http://yona.example.com/", 7)

	require.NoError(t, err)
	assert.Equal(t, "http://yona.example.com/git/bob/widgets.git", remoteURL)
	assert.Equal(t, "feature", branch)
	assert.Equal(t, "pr-7", localBranch)
}

func TestPlanCheckout_ErrorsWhenFromProjectMissing(t *testing.T) {
	_, _, _, err := planCheckout(map[string]interface{}{"fromBranch": "feature"}, "http://yona.example.com", 7)

	assert.Error(t, err)
}

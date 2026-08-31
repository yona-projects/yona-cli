package cmd

import (
	"fmt"
	"strings"

	"github.com/search5/yona-cli/internal/api"
	"github.com/search5/yona-cli/internal/config"
	"github.com/search5/yona-cli/internal/gitutil"
	"github.com/search5/yona-cli/internal/weburl"
	"github.com/spf13/cobra"
)

func newPRCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "풀 리퀘스트 조회/생성/수정/머지/리뷰",
	}
	cmd.AddCommand(newPRListCmd(ctx))
	cmd.AddCommand(newPRViewCmd(ctx))
	cmd.AddCommand(newPRCreateCmd(ctx))
	cmd.AddCommand(newPREditCmd(ctx))
	cmd.AddCommand(newPRMergeCmd(ctx))
	cmd.AddCommand(newPRCloseCmd(ctx))
	cmd.AddCommand(newPRReopenCmd(ctx))
	cmd.AddCommand(newPRDiffCmd(ctx))
	cmd.AddCommand(newPRCommentCmd(ctx))
	cmd.AddCommand(newPRReviewCmd(ctx))
	cmd.AddCommand(newPRCheckoutCmd(ctx))
	return cmd
}

func newPRListCmd(ctx *cmdContext) *cobra.Command {
	var repo, state, author, jsonFields string
	var limit int
	var web bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "풀 리퀘스트 목록을 조회한다",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			if web {
				server, err := config.ResolveServer(*ctx.server)
				if err != nil {
					return err
				}
				return gitutil.OpenInBrowser(weburl.PullRequestList(server, owner, project))
			}

			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			prs, err := client.ListPullRequests(cmd.Context(), owner, project, api.PullRequestListOptions{State: state, Author: author})
			if err != nil {
				return err
			}
			// 서버 GET .../pull-requests는 페이지네이션 없이 List<PullRequest> 전체를 반환하므로
			// -L/--limit은 클라이언트 사이드 슬라이싱으로 처리한다(계획 문서 지시대로).
			if limit > 0 && limit < len(prs) {
				prs = prs[:limit]
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, prs, jsonFields)
			}
			if len(prs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "풀 리퀘스트가 없습니다.")
				return nil
			}
			for _, pr := range prs {
				fmt.Fprintf(cmd.OutOrStdout(), "#%s\t%s\t%s\n", num(pr, "number"), str(pr, "state"), str(pr, "title"))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&state, "state", "", "상태로 필터링 (예: OPEN, CLOSED, MERGED)")
	cmd.Flags().StringVar(&author, "author", "", "작성자(contributor) 로그인ID로 필터링")
	cmd.Flags().IntVarP(&limit, "limit", "L", 0, "가져올 최대 개수 (클라이언트 사이드 슬라이싱, 생략 시 전체)")
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json number,title,state)")
	cmd.Flags().BoolVar(&web, "web", false, "API 호출 대신 풀 리퀘스트 목록 웹 페이지를 브라우저로 연다")
	return cmd
}

func newPRViewCmd(ctx *cmdContext) *cobra.Command {
	var repo, jsonFields string
	var web bool
	cmd := &cobra.Command{
		Use:   "view <number>",
		Short: "풀 리퀘스트 하나를 조회한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			number, err := parseNumberArg(args[0])
			if err != nil {
				return err
			}
			if web {
				server, err := config.ResolveServer(*ctx.server)
				if err != nil {
					return err
				}
				return gitutil.OpenInBrowser(weburl.PullRequest(server, owner, project, number))
			}

			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			pr, err := client.GetPullRequest(cmd.Context(), owner, project, number)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, pr, jsonFields)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "#%s %s\n상태: %s\n%s -> %s\n\n%s\n",
				num(pr, "number"), str(pr, "title"), str(pr, "state"), str(pr, "fromBranch"), str(pr, "toBranch"), str(pr, "body"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json number,title,state)")
	cmd.Flags().BoolVar(&web, "web", false, "API 호출 대신 풀 리퀘스트 웹 페이지를 브라우저로 연다")
	return cmd
}

func newPRCreateCmd(ctx *cmdContext) *cobra.Command {
	var repo, title, body, fromBranch, toBranch string
	var fromProjectID int64
	cmd := &cobra.Command{
		Use:   "create",
		Short: "새 풀 리퀘스트를 만든다",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			if title == "" || fromBranch == "" || toBranch == "" || fromProjectID == 0 {
				return fmt.Errorf("--title, --from-project-id, --from-branch, --to-branch는 모두 필수입니다")
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			pr, err := client.CreatePullRequest(cmd.Context(), owner, project, api.CreatePullRequestRequest{
				Title: title, Body: body, FromProjectID: fromProjectID, FromBranch: fromBranch, ToBranch: toBranch,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "풀 리퀘스트 #%s 생성됨: %s\n", num(pr, "number"), str(pr, "title"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상(to) 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&title, "title", "", "제목 (필수)")
	cmd.Flags().StringVar(&body, "body", "", "본문")
	cmd.Flags().Int64Var(&fromProjectID, "from-project-id", 0, "fork(from) 프로젝트의 숫자 ID (필수, 'yona project view'로 id 확인)")
	cmd.Flags().StringVar(&fromBranch, "from-branch", "", "fork(from) 브랜치 (필수)")
	cmd.Flags().StringVar(&toBranch, "to-branch", "", "대상(to) 브랜치 (필수)")
	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("from-project-id")
	_ = cmd.MarkFlagRequired("from-branch")
	_ = cmd.MarkFlagRequired("to-branch")
	return cmd
}

// newPREditCmd는 "gh pr edit" 대응 — yona-wiki P3-02 4라운드가 PATCH 어댑터를 추가한
// PullRequestApiController.update()를 그대로 감싼다(서비스 로직 자체는 Step5의 PUT부터 존재).
// title은 서버에서 non-null 필수값이라 --title을 생략하면 read-modify-write로 현재 값을 유지한다.
func newPREditCmd(ctx *cmdContext) *cobra.Command {
	var repo, title, body string
	cmd := &cobra.Command{
		Use:   "edit <number>",
		Short: "풀 리퀘스트 제목/본문을 수정한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			number, err := parseNumberArg(args[0])
			if err != nil {
				return err
			}
			if title == "" && body == "" {
				return fmt.Errorf("--title 또는 --body 중 하나는 지정해야 합니다")
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			req := api.UpdatePullRequestRequest{Title: title}
			if req.Title == "" {
				current, err := client.GetPullRequest(cmd.Context(), owner, project, number)
				if err != nil {
					return err
				}
				req.Title = rawStr(current, "title")
			}
			if body != "" {
				req.Body = &body
			}
			pr, err := client.UpdatePullRequest(cmd.Context(), owner, project, number, req)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "풀 리퀘스트 #%s을(를) 수정했습니다.\n", num(pr, "number"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&title, "title", "", "새 제목")
	cmd.Flags().StringVar(&body, "body", "", "새 본문")
	return cmd
}

func newPRMergeCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "merge <number>",
		Short: "풀 리퀘스트를 머지한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			number, err := parseNumberArg(args[0])
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			result, err := client.MergePullRequest(cmd.Context(), owner, project, number)
			if err != nil {
				return err
			}
			if conflicts, ok := result["conflicts"].(bool); ok && conflicts {
				fmt.Fprintf(cmd.OutOrStdout(), "풀 리퀘스트 #%d에 충돌이 있어 머지되지 않았습니다.\n", number)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "풀 리퀘스트 #%d을(를) 머지했습니다.\n", number)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	return cmd
}

// newPRCloseCmd/newPRReopenCmd는 "gh pr close"/"gh pr reopen" 대응 — yona-wiki P3-02 4라운드가
// 추가한 POST .../pull-requests/{number}/close|reopen을 그대로 감싼다(서버는 이미
// PullRequestController.changeState()로 양방향 지원 중이었다).
func newPRCloseCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "close <number>",
		Short: "풀 리퀘스트를 닫는다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			number, err := parseNumberArg(args[0])
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			if _, err := client.ClosePullRequest(cmd.Context(), owner, project, number); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "풀 리퀘스트 #%d을(를) 닫았습니다.\n", number)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	return cmd
}

func newPRReopenCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "reopen <number>",
		Short: "닫힌 풀 리퀘스트를 다시 연다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			number, err := parseNumberArg(args[0])
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			if _, err := client.ReopenPullRequest(cmd.Context(), owner, project, number); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "풀 리퀘스트 #%d을(를) 다시 열었습니다.\n", number)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	return cmd
}

// newPRDiffCmd는 "gh pr diff" 대응. 서버 응답(List<FileDiff>)의 신뢰도 관련 주의사항은
// internal/api/pr.go의 GetPullRequestDiff() 주석 참고 — pathA/pathB/changeType만 안전하게 쓰고
// 전체 원문은 --json으로만 노출한다.
func newPRDiffCmd(ctx *cmdContext) *cobra.Command {
	var repo, jsonFields string
	cmd := &cobra.Command{
		Use:   "diff <number>",
		Short: "풀 리퀘스트의 변경된 파일 목록을 보여준다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			number, err := parseNumberArg(args[0])
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			diffs, err := client.GetPullRequestDiff(cmd.Context(), owner, project, number)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, diffs, jsonFields)
			}
			if len(diffs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "변경된 파일이 없습니다.")
				return nil
			}
			for _, d := range diffs {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s -> %s\n", str(d, "changeType"), str(d, "pathA"), str(d, "pathB"))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json pathA,pathB,changeType)")
	return cmd
}

// newPRCommentCmd는 "gh pr comment" 대응 — yona PR은 PR 전체에 붙는 일반 댓글과 코드 라인 리뷰
// 댓글을 같은 서비스로 처리하며(internal/api/pr.go의 AddPullRequestComment() 주석 참고), 이
// 명령은 그중 PR 전체 댓글만 다룬다.
func newPRCommentCmd(ctx *cmdContext) *cobra.Command {
	var repo, body string
	cmd := &cobra.Command{
		Use:   "comment <number>",
		Short: "풀 리퀘스트에 댓글을 남긴다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			number, err := parseNumberArg(args[0])
			if err != nil {
				return err
			}
			if body == "" {
				return fmt.Errorf("--body는 필수입니다")
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			if _, err := client.AddPullRequestComment(cmd.Context(), owner, project, number, api.PullRequestCommentRequest{Body: body}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "풀 리퀘스트 #%d에 댓글을 남겼습니다.\n", number)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&body, "body", "", "댓글 내용 (필수)")
	_ = cmd.MarkFlagRequired("body")
	return cmd
}

// yona-wiki 계획 문서 주의사항 그대로: 이 서버의 addReviewer는 "리뷰어를 지정"하는 게 아니라
// 인증된 본인을 리뷰어로 자기등록하는 동작이다(PullRequestController.addReviewer 참고).
func newPRReviewCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "review <number>",
		Short: "본인을 해당 풀 리퀘스트의 리뷰어로 등록한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			number, err := parseNumberArg(args[0])
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			if err := client.AddReviewer(cmd.Context(), owner, project, number); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "풀 리퀘스트 #%d의 리뷰어로 등록되었습니다.\n", number)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	return cmd
}

// planCheckout은 "yona pr checkout"이 git fetch/checkout에 쓸 remote URL/브랜치/로컬 브랜치
// 이름을 계산하는 순수 함수다(테스트하기 쉽도록 git 실행과 분리) — yuna PR은 GitHub의
// refs/pull/N/head 같은 특수 ref가 아니라 실제 fromProject(owner/name)/fromBranch(평범한 브랜치)라
// clone URL만 계산하면 된다(TemplateHelper.getCloneUrl() 형식, 인증 없이 익명 fetch 기준).
func planCheckout(pr map[string]interface{}, baseURL string, number int64) (remoteURL, branch, localBranch string, err error) {
	fromProject, ok := pr["fromProject"].(map[string]interface{})
	if !ok {
		return "", "", "", fmt.Errorf("서버 응답에 fromProject 정보가 없어 이 풀 리퀘스트를 체크아웃할 수 없습니다")
	}
	fromOwner := rawStr(fromProject, "owner")
	fromName := rawStr(fromProject, "name")
	fromBranch := rawStr(pr, "fromBranch")
	if fromOwner == "" || fromName == "" || fromBranch == "" {
		return "", "", "", fmt.Errorf("서버 응답에서 fromProject/fromBranch를 확인할 수 없습니다")
	}
	remoteURL = strings.TrimRight(baseURL, "/") + "/" + fromOwner + "/" + fromName + ".git"
	localBranch = fmt.Sprintf("pr-%d", number)
	return remoteURL, fromBranch, localBranch, nil
}

// newPRCheckoutCmd는 "gh pr checkout" 대응. yona-wiki P3-02 계획 문서가 5라운드에서 재분류한
// 대로(서버 API 불필요, CLI 전용) "git fetch <fromProject clone URL> <fromBranch>" + "git
// checkout -B pr-<number> FETCH_HEAD"로 구현한다.
func newPRCheckoutCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "checkout <number>",
		Short: "풀 리퀘스트의 브랜치를 로컬로 체크아웃한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			number, err := parseNumberArg(args[0])
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			pr, err := client.GetPullRequest(cmd.Context(), owner, project, number)
			if err != nil {
				return err
			}
			remoteURL, branch, localBranch, err := planCheckout(pr, client.BaseURL, number)
			if err != nil {
				return err
			}
			if err := gitutil.FetchBranch(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), remoteURL, branch); err != nil {
				return err
			}
			if err := gitutil.CheckoutNewBranchFromFetchHead(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), localBranch); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "풀 리퀘스트 #%d을(를) 로컬 브랜치 %s로 체크아웃했습니다.\n", number, localBranch)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	return cmd
}

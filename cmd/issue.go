package cmd

import (
	"fmt"

	"github.com/yona-projects/yona-cli/internal/api"
	"github.com/yona-projects/yona-cli/internal/config"
	"github.com/yona-projects/yona-cli/internal/gitutil"
	"github.com/yona-projects/yona-cli/internal/weburl"
	"github.com/spf13/cobra"
)

func newIssueCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "이슈 조회/생성/수정/댓글/상태변경",
	}
	cmd.AddCommand(newIssueListCmd(ctx))
	cmd.AddCommand(newIssueViewCmd(ctx))
	cmd.AddCommand(newIssueCreateCmd(ctx))
	cmd.AddCommand(newIssueEditCmd(ctx))
	cmd.AddCommand(newIssueCommentCmd(ctx))
	cmd.AddCommand(newIssueCloseCmd(ctx))
	cmd.AddCommand(newIssueReopenCmd(ctx))
	cmd.AddCommand(newIssueTransferCmd(ctx))
	cmd.AddCommand(newIssueStatusCmd(ctx))
	return cmd
}

func newIssueListCmd(ctx *cmdContext) *cobra.Command {
	var repo, state, assignee, label, author, jsonFields string
	var limit int
	var web bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "이슈 목록을 조회한다",
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
				return gitutil.OpenInBrowser(weburl.IssueList(server, owner, project))
			}

			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			page, err := client.ListIssues(cmd.Context(), owner, project, api.IssueListOptions{
				State: state, Assignee: assignee, Label: label, Author: author, Limit: limit,
			})
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, page.Content, jsonFields)
			}
			if len(page.Content) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "이슈가 없습니다.")
				return nil
			}
			for _, issue := range page.Content {
				fmt.Fprintf(cmd.OutOrStdout(), "#%s\t%s\t%s\n", num(issue, "number"), str(issue, "state"), str(issue, "title"))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&state, "state", "", "상태로 필터링 (예: OPEN, CLOSED)")
	cmd.Flags().StringVar(&assignee, "assignee", "", "담당자 로그인ID로 필터링")
	cmd.Flags().StringVar(&label, "label", "", "라벨 이름으로 필터링")
	cmd.Flags().StringVar(&author, "author", "", "작성자 로그인ID로 필터링")
	cmd.Flags().IntVarP(&limit, "limit", "L", 0, "가져올 최대 개수 (서버 페이지네이션의 size 파라미터, 생략 시 서버 기본값)")
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json number,title,state)")
	cmd.Flags().BoolVar(&web, "web", false, "API 호출 대신 이슈 목록 웹 페이지를 브라우저로 연다")
	return cmd
}

func newIssueViewCmd(ctx *cmdContext) *cobra.Command {
	var repo, jsonFields string
	var web bool
	cmd := &cobra.Command{
		Use:   "view <number>",
		Short: "이슈 하나를 조회한다",
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
				return gitutil.OpenInBrowser(weburl.Issue(server, owner, project, number))
			}

			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			issue, err := client.GetIssue(cmd.Context(), owner, project, number)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, issue, jsonFields)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "#%s %s\n상태: %s\n\n%s\n", num(issue, "number"), str(issue, "title"), str(issue, "state"), str(issue, "body"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json number,title,state)")
	cmd.Flags().BoolVar(&web, "web", false, "API 호출 대신 이슈 웹 페이지를 브라우저로 연다")
	return cmd
}

// resolveIssueAssigneeLabelFlags는 issue/PR create/edit 명령이 공통으로 쓰는 "--assignee
// <loginId>"/"--label <name>(반복 가능)" 플래그를 서버가 요구하는 숫자 assigneeId/labelIds로
// 변환한다(yona-wiki P3-02 13라운드/TASK-0430 — 서버 REST API엔 필드가 이미 있었지만 CLI에 배선이
// 아예 없던 갭 해소).
func resolveAssigneeFlag(cmd *cobra.Command, client *api.Client, owner, project, loginID string) (*int64, error) {
	id, err := client.ResolveAssigneeID(cmd.Context(), owner, project, loginID)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func resolveLabelFlags(cmd *cobra.Command, client *api.Client, owner, project string, names []string) ([]int64, error) {
	ids := make([]int64, 0, len(names))
	for _, name := range names {
		id, err := client.ResolveLabelID(cmd.Context(), owner, project, name)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func newIssueCreateCmd(ctx *cmdContext) *cobra.Command {
	var repo, title, body, assignee string
	var labels []string
	var draft bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "새 이슈를 만든다",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			if title == "" {
				return fmt.Errorf("--title은 필수입니다")
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			req := api.CreateIssueRequest{Title: title, Body: body, IsDraft: draft}
			if assignee != "" {
				id, err := resolveAssigneeFlag(cmd, client, owner, project, assignee)
				if err != nil {
					return err
				}
				req.AssigneeID = id
			}
			if len(labels) > 0 {
				ids, err := resolveLabelFlags(cmd, client, owner, project, labels)
				if err != nil {
					return err
				}
				req.LabelIDs = ids
			}
			issue, err := client.CreateIssue(cmd.Context(), owner, project, req)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "이슈 #%s 생성됨: %s\n", num(issue, "number"), str(issue, "title"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&title, "title", "", "이슈 제목 (필수)")
	cmd.Flags().StringVar(&body, "body", "", "이슈 본문")
	cmd.Flags().StringVar(&assignee, "assignee", "", "담당자 로그인ID")
	cmd.Flags().StringArrayVar(&labels, "label", nil, "라벨 이름 (반복 지정 가능)")
	cmd.Flags().BoolVar(&draft, "draft", false, "초안(DRAFT)으로 생성")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

// newIssueEditCmd는 "gh issue edit" 대응 — REST API는 Step4부터 PATCH를 이미 지원했지만
// (IssueRestApiController.update()) CLI 쪽 명령이 없었다. --title/--body를 생략하면
// read-modify-write로 현재 값을 그대로 유지한다(UpdateIssueRequest의 title/body가 서버에서
// non-null 필수값이라 부분 수정이라도 전체를 채워 보내야 한다).
//
// yona-wiki P3-02 13라운드(TASK-0430) — IssueServiceImpl.updateIssue()는 title/body와 달리
// assignee/milestone/labels를 "요청에 값이 없으면 기존 값 유지"가 아니라 "요청에 값이 없으면
// null/빈 값으로 덮어쓴다"(전체 교체 시맨틱, 웹 폼이 항상 현재 값을 hidden input으로 재제출하는
// 것을 전제로 설계됨). CLI가 이 필드들을 아예 보내지 않고 있어 지금까지 `yona issue edit`을
// 한 번이라도 호출하면 그 이슈의 담당자/마일스톤/라벨이 전부 조용히 지워지는 데이터 손실
// 버그였다 — --assignee/--label을 새로 추가하면서 title/body와 동일한 read-modify-write
// 패턴으로 세 필드 모두 보존하도록 함께 고쳤다.
func newIssueEditCmd(ctx *cmdContext) *cobra.Command {
	var repo, title, body, assignee string
	var removeAssignee bool
	var labels []string
	cmd := &cobra.Command{
		Use:   "edit <number>",
		Short: "이슈 제목/본문/담당자/라벨을 수정한다",
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
			if title == "" && body == "" && assignee == "" && !removeAssignee && len(labels) == 0 {
				return fmt.Errorf("--title, --body, --assignee, --remove-assignee, --label 중 하나는 지정해야 합니다")
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			current, err := client.GetIssue(cmd.Context(), owner, project, number)
			if err != nil {
				return err
			}
			newTitle, newBody := title, body
			if newTitle == "" {
				newTitle = rawStr(current, "title")
			}
			if newBody == "" {
				newBody = rawStr(current, "body")
			}
			req := api.UpdateIssueRequest{Title: newTitle, Body: newBody}

			// milestoneId: 이 명령엔 아직 마일스톤 변경 플래그가 없으므로 항상 현재 값을 보존한다.
			if milestoneID, ok := rawInt64(current, "milestoneId"); ok {
				req.MilestoneID = &milestoneID
			}

			// assigneeId: --remove-assignee면 명시적으로 비움(nil), --assignee면 새로 지정,
			// 둘 다 없으면 현재 담당자를 그대로 보존한다.
			switch {
			case removeAssignee:
				// req.AssigneeID는 nil 그대로 둔다 (담당자 해제).
			case assignee != "":
				id, err := resolveAssigneeFlag(cmd, client, owner, project, assignee)
				if err != nil {
					return err
				}
				req.AssigneeID = id
			default:
				if id, ok := currentAssigneeUserID(current); ok {
					req.AssigneeID = &id
				}
			}

			// labelIds: --label을 하나라도 지정하면 그 집합으로 완전히 교체하고(서버가 부분
			// 추가/제거가 아닌 전체 교체만 지원), 지정하지 않으면 현재 라벨 집합을 그대로 보존한다.
			if len(labels) > 0 {
				ids, err := resolveLabelFlags(cmd, client, owner, project, labels)
				if err != nil {
					return err
				}
				req.LabelIDs = ids
			} else {
				req.LabelIDs = currentLabelIDs(current)
			}

			issue, err := client.UpdateIssue(cmd.Context(), owner, project, number, req)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "이슈 #%s을(를) 수정했습니다.\n", num(issue, "number"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&title, "title", "", "새 제목")
	cmd.Flags().StringVar(&body, "body", "", "새 본문")
	cmd.Flags().StringVar(&assignee, "assignee", "", "담당자로 지정할 로그인ID")
	cmd.Flags().BoolVar(&removeAssignee, "remove-assignee", false, "담당자 지정을 해제한다")
	cmd.Flags().StringArrayVar(&labels, "label", nil, "라벨 이름 (반복 지정 가능 — 지정하면 기존 라벨 전체를 이 집합으로 교체)")
	return cmd
}

func newIssueCommentCmd(ctx *cmdContext) *cobra.Command {
	var repo, body string
	cmd := &cobra.Command{
		Use:   "comment <number>",
		Short: "이슈에 댓글을 남긴다",
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
			if _, err := client.AddIssueComment(cmd.Context(), owner, project, number, api.CommentRequest{Contents: body}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "이슈 #%d에 댓글을 남겼습니다.\n", number)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&body, "body", "", "댓글 내용 (필수)")
	_ = cmd.MarkFlagRequired("body")
	return cmd
}

func newIssueCloseCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "close <number>",
		Short: "이슈를 닫는다",
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
			if _, err := client.CloseIssue(cmd.Context(), owner, project, number); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "이슈 #%d을(를) 닫았습니다.\n", number)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	return cmd
}

// newIssueReopenCmd는 "gh issue reopen" 대응 — yona-wiki P3-02 4라운드가 추가한
// POST .../issues/{number}/reopen을 그대로 감싼다.
func newIssueReopenCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "reopen <number>",
		Short: "닫힌 이슈를 다시 연다",
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
			if _, err := client.ReopenIssue(cmd.Context(), owner, project, number); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "이슈 #%d을(를) 다시 열었습니다.\n", number)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	return cmd
}

// newIssueTransferCmd는 "gh issue transfer" 대응 — yona-wiki P3-02 4라운드가 추가한
// POST .../issues/{number}/transfer(대상을 owner/project 이름으로 받아 서버가 내부에서
// 숫자 id로 resolve)를 그대로 감싼다.
func newIssueTransferCmd(ctx *cmdContext) *cobra.Command {
	var repo, target string
	cmd := &cobra.Command{
		Use:   "transfer <number>",
		Short: "이슈를 다른 프로젝트로 옮긴다",
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
			if target == "" {
				return fmt.Errorf(`--to는 필수입니다 ("owner/project" 형식)`)
			}
			targetOwner, targetProject, err := parseRepo(target)
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			if _, err := client.TransferIssue(cmd.Context(), owner, project, number, api.TransferIssueRequest{
				TargetOwner: targetOwner, TargetProject: targetProject,
			}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "이슈 #%d을(를) %s으로 옮겼습니다.\n", number, target)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&target, "to", "", `옮길 대상 프로젝트, "owner/project" 형식 (필수)`)
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

// newIssueStatusCmd는 "gh issue status" 최소 버전 대응 — yona-wiki P3-02 4라운드가 추가한
// GET /api/v1/user/issues/status(담당/작성 이슈 개수·목록)를 그대로 감싼다. mentioned/favorite/
// shared 필터와 페이지네이션 확장은 서버 자체가 최소 버전이라 다음 라운드로 이월된 상태다.
func newIssueStatusCmd(ctx *cmdContext) *cobra.Command {
	var jsonFields string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "내가 담당하거나 작성한 이슈 현황을 보여준다",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			status, err := client.GetIssueStatus(cmd.Context())
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, status, jsonFields)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "담당 중인 이슈 (열림 %d / 닫힘 %d)\n", status.Assigned.OpenCount, status.Assigned.ClosedCount)
			for _, issue := range status.Assigned.Items {
				fmt.Fprintf(cmd.OutOrStdout(), "  #%s\t%s\n", num(issue, "number"), str(issue, "title"))
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n작성한 이슈 (열림 %d / 닫힘 %d)\n", status.Created.OpenCount, status.Created.ClosedCount)
			for _, issue := range status.Created.Items {
				fmt.Fprintf(cmd.OutOrStdout(), "  #%s\t%s\n", num(issue, "number"), str(issue, "title"))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json assigned,created)")
	return cmd
}

package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/yona-projects/yona-cli/internal/api"
)

// newWikiCmd는 yona 서버의 프로젝트 위키(P3-42, WikiRestApiController.kt)를 감싼다. 위키
// 페이지는 DB가 아니라 그 프로젝트의 별도 bare git 저장소("<owner>/<project>.wiki.git") 안의
// 마크다운 파일이라, 이 명령들도 "제목(=파일 경로, 슬래시로 하위 경로 표현 가능)"을 그대로
// 리소스 식별자로 쓴다 — gh CLI에는 위키 명령이 없어(GitHub 위키는 CLI 미지원) 이 세션 표준
// 관례대로 tag/label 명령과 동일한 형태로 새로 설계했다.
func newWikiCmd(ctx *cmdContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wiki",
		Short: "프로젝트 위키 페이지 조회/생성/수정/삭제/히스토리",
	}
	cmd.AddCommand(newWikiListCmd(ctx))
	cmd.AddCommand(newWikiViewCmd(ctx))
	cmd.AddCommand(newWikiCreateCmd(ctx))
	cmd.AddCommand(newWikiEditCmd(ctx))
	cmd.AddCommand(newWikiDeleteCmd(ctx))
	cmd.AddCommand(newWikiHistoryCmd(ctx))
	cmd.AddCommand(newWikiDiffCmd(ctx))
	cmd.AddCommand(newWikiCompareCmd(ctx))
	return cmd
}

// readContentInput은 --content(리터럴 문자열)와 --content-file(파일 경로, "-"는 표준입력) 중
// 정확히 하나만 허용한다 — yona api 명령의 --input 관례(cmd/api.go)와 동일하다.
func readContentInput(cmd *cobra.Command, content, contentFile string) (string, error) {
	hasContent := cmd.Flags().Changed("content")
	hasFile := contentFile != ""
	if hasContent && hasFile {
		return "", fmt.Errorf("--content와 --content-file은 함께 쓸 수 없습니다")
	}
	if hasFile {
		if contentFile == "-" {
			data, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return "", fmt.Errorf("표준입력을 읽을 수 없습니다: %w", err)
			}
			return string(data), nil
		}
		data, err := os.ReadFile(contentFile)
		if err != nil {
			return "", fmt.Errorf("--content-file을 읽을 수 없습니다(%s): %w", contentFile, err)
		}
		return string(data), nil
	}
	return content, nil
}

func newWikiListCmd(ctx *cmdContext) *cobra.Command {
	var repo, query, jsonFields string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "위키 페이지 목록을 조회한다(--query로 제목 검색, P3-42 8번)",
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			pages, err := client.ListWikiPages(cmd.Context(), owner, project, query)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, pages, jsonFields)
			}
			if len(pages) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "위키 페이지가 없습니다.")
				return nil
			}
			for _, p := range pages {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", str(p, "title"), str(p, "lastCommitMessage"))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&query, "query", "", "제목 부분일치 검색어(생략하면 전체 목록)")
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json title,updatedAt)")
	return cmd
}

func newWikiViewCmd(ctx *cmdContext) *cobra.Command {
	var repo, jsonFields string
	cmd := &cobra.Command{
		Use:   "view <title>",
		Short: `위키 페이지 원문을 조회한다("Guides/Setup"처럼 슬래시로 하위 경로 표현 가능, P3-42 7번)`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			page, err := client.GetWikiPage(cmd.Context(), owner, project, args[0])
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, page, jsonFields)
			}
			fmt.Fprintln(cmd.OutOrStdout(), str(page, "content"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json title,content,revision)")
	return cmd
}

func newWikiCreateCmd(ctx *cmdContext) *cobra.Command {
	var repo, content, contentFile, message string
	cmd := &cobra.Command{
		Use:   "create <title>",
		Short: "새 위키 페이지를 만든다(P3-42 1번)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			body, err := readContentInput(cmd, content, contentFile)
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			page, err := client.CreateWikiPage(cmd.Context(), owner, project, api.CreateWikiPageRequest{
				Title: args[0], Content: body, Message: message,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "위키 페이지 %q 생성됨 (리비전: %s)\n", str(page, "title"), str(page, "revision"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&content, "content", "", "페이지 본문(마크다운) 리터럴 문자열")
	cmd.Flags().StringVar(&contentFile, "content-file", "", `페이지 본문을 파일에서 읽는다("-"는 표준입력). --content와 함께 쓸 수 없다`)
	cmd.Flags().StringVar(&message, "message", "", `커밋 메시지(생략하면 "Create <title>", P3-42 4번)`)
	return cmd
}

func newWikiEditCmd(ctx *cmdContext) *cobra.Command {
	var repo, newTitle, content, contentFile, message string
	cmd := &cobra.Command{
		Use:   "edit <title>",
		Short: "위키 페이지를 수정한다(--new-title을 주면 이름변경까지 한 커밋으로 반영, P3-42 1번)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			body, err := readContentInput(cmd, content, contentFile)
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("content") && contentFile == "" {
				return fmt.Errorf("--content 또는 --content-file 중 하나는 지정해야 합니다")
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			page, err := client.UpdateWikiPage(cmd.Context(), owner, project, args[0], api.UpdateWikiPageRequest{
				NewTitle: newTitle, Content: body, Message: message,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "위키 페이지 %q 수정됨 (리비전: %s)\n", str(page, "title"), str(page, "revision"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&newTitle, "new-title", "", "새 제목(이름변경, 생략하면 제목 유지)")
	cmd.Flags().StringVar(&content, "content", "", "새 본문(마크다운) 리터럴 문자열")
	cmd.Flags().StringVar(&contentFile, "content-file", "", `새 본문을 파일에서 읽는다("-"는 표준입력). --content와 함께 쓸 수 없다`)
	cmd.Flags().StringVar(&message, "message", "", `커밋 메시지(생략하면 자동 생성, P3-42 4번)`)
	return cmd
}

func newWikiDeleteCmd(ctx *cmdContext) *cobra.Command {
	var repo, message string
	cmd := &cobra.Command{
		Use:   "delete <title>",
		Short: "위키 페이지를 삭제한다",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			if err := client.DeleteWikiPage(cmd.Context(), owner, project, args[0], message); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "위키 페이지 %q을(를) 삭제했습니다.\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().StringVar(&message, "message", "", `커밋 메시지(생략하면 "Delete <title>")`)
	return cmd
}

func newWikiHistoryCmd(ctx *cmdContext) *cobra.Command {
	var repo, jsonFields string
	var pageNum, pageSize int
	cmd := &cobra.Command{
		Use:   "history <title>",
		Short: "위키 페이지의 리비전(커밋) 목록을 최신순으로 조회한다(P3-42 3번)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			revisions, err := client.ListWikiHistory(cmd.Context(), owner, project, args[0], pageNum, pageSize)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("json") {
				return printJSON(cmd, revisions, jsonFields)
			}
			if len(revisions) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "히스토리가 없습니다.")
				return nil
			}
			for _, r := range revisions {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", str(r, "shortId"), str(r, "authorName"), str(r, "shortMessage"))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	cmd.Flags().IntVar(&pageNum, "page", 0, "0부터 시작하는 페이지 번호")
	cmd.Flags().IntVar(&pageSize, "page-size", 20, "페이지당 리비전 개수")
	cmd.Flags().StringVar(&jsonFields, "json", "", "콤마로 구분한 필드만 뽑아 JSON으로 출력 (예: --json id,shortMessage,authorName)")
	return cmd
}

func newWikiDiffCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "diff <commit-id> <title>",
		Short: "그 리비전(커밋 vs 부모)이 그 위키 페이지에 반영한 변경 내용을 unified diff로 보여준다",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			result, err := client.GetWikiDiff(cmd.Context(), owner, project, args[0], args[1])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), str(result, "patch"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	return cmd
}

func newWikiCompareCmd(ctx *cmdContext) *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "compare <rev-a> <rev-b> <title>",
		Short: "임의의 두 리비전 사이에서 그 위키 페이지가 어떻게 달라졌는지 unified diff로 보여준다",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, project, err := resolveRepo(cmd, repo)
			if err != nil {
				return err
			}
			client, err := ctx.newClient()
			if err != nil {
				return err
			}
			result, err := client.CompareWikiRevisions(cmd.Context(), owner, project, args[0], args[1], args[2])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), str(result, "patch"))
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", `대상 프로젝트, "owner/project" 형식 (생략 시 현재 디렉터리의 git origin remote로 자동감지)`)
	return cmd
}

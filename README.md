# yona-cli

[![Release](https://img.shields.io/github/v/release/yona-projects/yona-cli)](https://github.com/yona-projects/yona-cli/releases)

[yona](https://github.com/yona-projects/yona) 서버의 REST API를 감싸는 커맨드라인 도구다.
GitHub CLI(`gh`)와 사용법이 같다.

## 설치

```bash
curl -fsSL https://raw.githubusercontent.com/yona-projects/yona-cli/main/install.sh | sh
```

Linux/macOS(amd64/arm64) 바이너리를 [GitHub Releases](https://github.com/yona-projects/yona-cli/releases)에서 받아 `~/.local/bin`에 설치한다. 설치 경로는 `YONA_INSTALL_DIR`로, 버전은 `YONA_VERSION`으로 지정할 수 있다. Windows는 Releases 페이지에서 `.zip`을 내려받는다.

소스에서 직접 빌드하려면:

```bash
go build -o bin/yona .
```

Homebrew/Scoop/`.deb`/`.rpm` 패키지는 아직 지원하지 않는다.

## 사용법

```bash
# 로그인 (yona 웹 UI에서 발급받은 Personal Access Token 필요)
yona auth login --server https://yona.example.com

# 로그인 상태 확인
yona auth status

# 이슈/PR/프로젝트 조회 (--repo 생략 시 현재 디렉터리의 git origin remote로 자동감지)
yona project list acme
yona project view acme/widgets
yona issue list
yona issue view 42
yona pr list

# 로그아웃
yona auth logout
```

## 인증

`yona auth login`에는 웹 UI에서 미리 발급받은 토큰이 필요하다. 다음 중 하나를 발급해 입력한다.

- 레거시 전권 토큰(`/user/editform/token_reset`)
- 모든 스코프 그룹을 write로 선택한 Fine-grained 토큰(`/user/editform/tokens`)

CI/봇/서드파티 연동처럼 권한을 좁히고 싶을 때는 저장소/권한 범위를 제한한 Fine-grained 토큰을 발급받아 `--token`으로 그때그때 넘긴다. 이 값은 설정 파일에 저장되지 않고 해당 호출에만 적용된다.

```bash
yona issue list --repo acme/widgets --token <제한된 토큰>
```

서버와 토큰은 다음 순서로 결정된다.

1. `--server`/`--token` 플래그
2. `YONA_HOST`/`YONA_TOKEN` 환경변수
3. `~/.config/yona-cli/config.yml`에 저장된 값(`yona auth login`이 기록)

## 공통 플래그

`gh` CLI와 동일한 관례를 따른다.

- **`--repo`/`-R`**: 생략하면 현재 디렉터리의 `git remote get-url origin`으로 owner/project를 자동감지한다(호스트 뒤 마지막 두 경로 세그먼트를 owner/project로 사용). git 저장소 밖이면 오류를 반환한다.
- **`--json <fields>`**: 콤마로 구분한 필드만 JSON으로 출력한다(예: `--json number,title,state`). 필드 목록은 필수다.
- **`-L, --limit <N>`**: 결과 개수를 제한한다.
- **`--web`**: API 호출 대신 해당 리소스의 웹 페이지를 브라우저로 연다(`view`/`list` 계열).

## 명령어

### `yona auth`

| 명령 | 설명 |
|---|---|
| `yona auth login --server <URL> [--token <값>]` | 로그인 정보 저장 |
| `yona auth logout` | 로그인 정보 삭제 |
| `yona auth status` | 로그인 상태 표시(토큰은 마스킹) |

### `yona server`

여러 yona 서버를 오갈 때 쓴다. `use`는 이미 로그인된 서버로 전환만 하며 재로그인을 요구하지 않는다.

| 명령 | 설명 |
|---|---|
| `yona server list` | 로그인된 서버 목록(현재 서버는 `*` 표시) |
| `yona server use <서버 URL>` | 이미 로그인된 서버로 전환 |

### `yona browse`

| 명령 | 설명 |
|---|---|
| `yona browse [--repo <owner/project>]` | 프로젝트 홈 열기 |
| `yona browse issue <number> [--repo ...]` | 이슈 상세 페이지 열기 |
| `yona browse pr <number> [--repo ...]` | PR 상세 페이지 열기 |

### `yona issue`

| 명령 | 설명 |
|---|---|
| `yona issue list [-R <owner/project>] [--state] [--assignee] [--label] [--author] [-L N] [--json fields] [--web]` | 목록 |
| `yona issue view <number> [-R ...] [--json fields] [--web]` | 조회 |
| `yona issue create -R <owner/project> --title <제목> [--body] [--draft]` | 생성 |
| `yona issue edit <number> [-R ...] [--title] [--body]` | 제목/본문 수정(생략한 필드는 기존 값 유지) |
| `yona issue comment <number> [-R ...] --body <내용>` | 댓글 작성 |
| `yona issue close <number> [-R ...]` | 닫기 |
| `yona issue reopen <number> [-R ...]` | 다시 열기 |
| `yona issue transfer <number> [-R ...] --to <owner/project>` | 다른 프로젝트로 이동 |
| `yona issue status [--state] [--filter] [--page] [--json fields]` | 담당/작성/댓글단/멘션된/즐겨찾기한/공유받은 이슈 현황 |

### `yona pr`

| 명령 | 설명 |
|---|---|
| `yona pr list [-R <owner/project>] [--state] [--author] [-L N] [--json fields] [--web]` | 목록 |
| `yona pr view <number> [-R ...] [--json fields] [--web]` | 조회 |
| `yona pr create -R <owner/project> --title <제목> --from <owner/project> --from-branch <브랜치> --to-branch <브랜치>` | 생성(`--from`은 fork 프로젝트를 "owner/project" 형식으로 지정) |
| `yona pr edit <number> [-R ...] [--title] [--body]` | 제목/본문 수정(생략한 필드는 기존 값 유지) |
| `yona pr merge <number> [-R ...]` | 머지 |
| `yona pr close <number> [-R ...]` | 닫기 |
| `yona pr reopen <number> [-R ...]` | 다시 열기 |
| `yona pr diff <number> [-R ...] [--json fields]` | 변경된 파일 목록(pathA/pathB/changeType) |
| `yona pr comment <number> [-R ...] --body <내용>` | PR 전체에 댓글 작성 |
| `yona pr review <number> [-R ...] [--remove] [--approve\|--request-changes\|--comment] [--body]` | 본인을 리뷰어로 등록/취소하거나 Approve/Request changes/Comment 판정 제출 |
| `yona pr checkout <number> [-R ...]` | fromProject/fromBranch로 로컬 브랜치 생성 |

### `yona project`

| 명령 | 설명 |
|---|---|
| `yona project list <owner> [-L N] [--json fields]` | owner 아래 프로젝트 목록 |
| `yona project view <owner/project> [--json fields] [--web]` | 프로젝트 조회 |
| `yona project create <owner/name> [--overview] [--scope PUBLIC\|PRIVATE] [--vcs GIT]` | 생성(세션 로그인/레거시 전권 토큰만 가능) |
| `yona project fork <owner/project> [--to-owner <대상>] [--to-name <이름>]` | 자신의 계정 또는 조직 아래로 fork |
| `yona project edit <owner/project> [--name] [--overview] [--scope] [--default-branch]` | 설정 수정(생략한 필드는 기존 값 유지) |
| `yona project delete <owner/project> --yes` | 삭제(되돌릴 수 없음) |

### `yona label`

| 명령 | 설명 |
|---|---|
| `yona label list [-R <owner/project>] [--json fields]` | 목록 |
| `yona label create [-R ...] --name --color --category [--exclusive]` | 생성 |
| `yona label edit <id> [-R ...] --name --color --category-id` | 수정 |
| `yona label delete <id> [-R ...]` | 삭제 |

### `yona tag`

| 명령 | 설명 |
|---|---|
| `yona tag list [-R <owner/project>] [--json fields]` | 목록(lightweight/annotated 구분) |
| `yona tag create <name> [-R ...] [--target] [--message]` | 생성(`--message`를 주면 annotated 태그, 생략하면 lightweight 태그) |
| `yona tag delete <name> [-R ...]` | 삭제(매니저/조직관리자 권한 필요) |

### `yona wiki`

프로젝트 위키 페이지를 다룬다. 제목은 슬래시로 하위 경로를 표현할 수 있다(예: `Guides/Setup`).

| 명령 | 설명 |
|---|---|
| `yona wiki list [-R <owner/project>] [--query] [--json fields]` | 페이지 목록(`--query`로 제목 검색) |
| `yona wiki view <title> [-R ...] [--json fields]` | 페이지 원문 조회 |
| `yona wiki create <title> [-R ...] (--content <텍스트>\|--content-file <파일>) [--message]` | 페이지 생성 |
| `yona wiki edit <title> [-R ...] [--new-title] (--content\|--content-file) [--message]` | 페이지 수정(이름변경 포함) |
| `yona wiki delete <title> [-R ...] [--message]` | 페이지 삭제 |
| `yona wiki history <title> [-R ...] [--page] [--page-size] [--json fields]` | 리비전 목록 |
| `yona wiki diff <commit-id> <title> [-R ...]` | 해당 리비전의 변경 내용 |
| `yona wiki compare <rev-a> <rev-b> <title> [-R ...]` | 두 리비전 사이의 변경 내용 |

`--content-file`은 `-`를 주면 표준입력에서 읽는다(`--content`와 동시에 쓸 수 없음).

### `yona search`

| 명령 | 설명 |
|---|---|
| `yona search issues <query> [--page] [--size] [--json fields]` | 이슈 검색 |
| `yona search prs <query> [--page] [--size] [--json fields]` | 풀 리퀘스트 검색 |
| `yona search projects <query> [--page] [--size] [--json fields]` | 프로젝트 검색 |

### `yona org`

| 명령 | 설명 |
|---|---|
| `yona org list [--filter] [--page] [--json fields]` | 조직 목록 |
| `yona org view <name> [--json fields]` | 조직 조회(소속 프로젝트 포함) |

### `yona admin`

| 명령 | 설명 | 상태 |
|---|---|---|
| `yona admin backup export [-o <파일>]` | 전체 백업 다운로드(사이트 매니저 전용) | 구현됨 |
| `yona admin backup import <파일>` | 전체 복원(기존 데이터 완전 교체) | 구현됨 |
| `yona admin webhook create --repo <owner/project> --url <URL> [--secret][--git-push][--type]` | 웹훅 생성 | 구현됨 |
| `yona admin webhook delete <id> --repo <owner/project>` | 웹훅 삭제 | 구현됨 |
| `yona admin webhook list --repo <owner/project>` | 웹훅 목록 | 미구현 |
| `yona admin permission add <loginId> --repo <owner/project>` | 멤버 추가 | 구현됨 |
| `yona admin permission update-role <userId> <roleId> --repo <owner/project>` | 역할 변경 | 구현됨 |
| `yona admin permission remove <userId> --repo <owner/project>` | 멤버 제거 | 구현됨 |
| `yona admin permission list --repo <owner/project>` | 멤버/권한 목록 | 미구현 |

### `yona api`

CLI가 감싸지 않은 엔드포인트를 직접 호출한다.

```bash
yona api /api/v1/projects/acme
yona api -X POST -f title=hello -f body=world /api/v1/projects/acme/widgets/issues
yona api -X DELETE /api/v1/projects/acme/widgets/issues/1
echo '{"title":"raw body"}' | yona api -X POST --input - /api/v1/projects/acme/widgets/issues
```

### `yona config`

| 명령 | 설명 |
|---|---|
| `yona config get <key>` | 설정값 조회 |
| `yona config set <key> <value>` | 설정값 저장 |
| `yona config list` | 지원 키와 현재 값 목록 |

지원 키: `browser`(`--web`/`browse` 계열이 열 브라우저 명령, `BROWSER` 환경변수가 우선한다).

### `yona alias`

| 명령 | 설명 |
|---|---|
| `yona alias set <name> <expansion>` | 별칭 등록(예: `yona alias set pv "pr view"`) |
| `yona alias list` | 등록된 별칭 목록 |
| `yona alias delete <name>` | 별칭 삭제 |

다른 yona 하위 명령으로만 확장되는 별칭만 지원한다(셸 명령 별칭은 지원하지 않는다). 기존 최상위 명령과 이름이 겹치는 별칭은 등록할 수 없다.

### 기타

- `yona --version` — 버전 출력
- `yona completion [bash|zsh|fish|powershell]` — 쉘 자동완성 스크립트 생성

## 개발

```
.
├── main.go              # 진입점
├── install.sh            # 설치 스크립트
├── scripts/release.sh     # 릴리즈 빌드 및 배포
├── cmd/                  # Cobra 명령 트리
├── internal/api/          # yona REST API HTTP 클라이언트
├── internal/gitutil/      # 로컬 git 연동
├── internal/weburl/       # 웹 UI 페이지 URL 계산
└── internal/config/       # 설정 파일(~/.config/yona-cli/config.yml) 로드/저장
```

```bash
go test ./...
```

실제 yona 서버 없이 `net/http/httptest`로 서버 응답을 목킹해 검증한다.

## 알려진 제한사항

- `project create`는 Fine-grained 토큰으로 인증할 수 없다(세션 로그인 또는 레거시 전권 토큰 필요).
- `yona admin webhook list`, `yona admin permission list`는 서버에 대응 API가 없어 미구현이다.
- SSH 인증, `runner`/`workflow`, `mcp serve`는 CLI 명령으로 제공하지 않는다.
- Homebrew/Scoop/`.deb`/`.rpm` 패키지 배포는 아직 지원하지 않는다.

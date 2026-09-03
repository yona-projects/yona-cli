# yona-cli

`yona-cli`는 [yona](https://github.com/yona-projects/yona) 서버(REST API)를 감싸는 커맨드라인 도구다.
GitHub CLI(`gh`)와 동일한 컨셉/스택(Go + Cobra)으로 만들었다 — 설치 직후 바로 실행돼야 하므로
JVM 콜드스타트가 있는 Kotlin은 제외하고 Go를 채택했다(자세한 배경은 yona 저장소의
`docs/yona-wiki/plans/p3-02-cli-and-rest-api.md` 참고).

## 설치

```bash
go build -o bin/yona .
```

(추후 `goreleaser`로 GitHub Releases/Homebrew/Scoop/`.deb`/`.rpm` 배포 예정 — 이번 라운드
범위 밖)

## 빠른 시작

```bash
# 1. 로그인 (yona 웹 UI에서 미리 발급받은 Personal Access Token 필요)
yona auth login --server https://yona.example.com
# Personal Access Token: (입력, 화면에 표시되지 않음)

# 2. 로그인 상태 확인
yona auth status

# 3. 이슈/PR/프로젝트 조회 (--repo 생략 시 현재 디렉터리의 git origin remote로 자동감지)
yona project list acme
yona project view acme/widgets
yona issue list
yona issue view 42
yona pr list

# 4. 로그아웃
yona auth logout
```

### 인증 토큰의 기본 스코프

`yona auth login`은 **본인 계정이 가진 전체 권한**(전체 저장소 + 모든 스코프 그룹 write)을
전제로 한다 — 웹 세션 로그인과 동등한 권한이라는 뜻이다. yona 서버 쪽에 OAuth 유사 로그인
플로우가 없으므로, 실제로는 다음 둘 중 하나를 웹 UI에서 미리 발급받아 이 명령에 입력한다.

- 레거시 전권 토큰(`/user/editform/token_reset`)
- 모든 스코프 그룹을 write로 선택한 Fine-grained 토큰(`/user/editform/tokens`)

CI/봇/서드파티 연동처럼 유출 피해를 줄이고 싶을 때는, 웹 UI에서 저장소/권한 범위를 좁힌
Fine-grained 토큰을 별도로 발급한 뒤 `--token <값>`으로 그때그때 넘겨 쓴다(설정 파일에
저장하지 않고 그 호출에만 적용됨).

```bash
yona issue list --repo acme/widgets --token <제한된 토큰>
```

## 서버/토큰 결정 순서

1. `--server`/`--token` 플래그
2. `YONA_HOST`/`YONA_TOKEN` 환경변수
3. `~/.config/yona-cli/config.yml`에 저장된 값(`yona auth login`이 기록)

## `--repo`/`-R` 자동감지, `--json`, `-L/--limit`, `--web` 공통 관례

`gh` CLI와 동일한 관례를 따른다.

- **`--repo`/`-R`**: 생략하면 현재 디렉터리에서 `git remote get-url origin`을 실행해 owner/project를
  자동감지한다(yona clone URL 형식은 `TemplateHelper.getCloneUrl()` 참고 — 호스트 뒤 마지막 두
  경로 세그먼트를 owner/project로 취급). git이 없거나 저장소 밖이면 명시적 `--repo` 오류로 폴백한다.
- **`--json <fields>`**: 콤마로 구분한 필드만 뽑아 JSON으로 출력한다(예: `--json number,title,state`).
  값 없이 `--json`만 쓰면 오류다(필드 목록이 필수). 예전의 불리언 스위치(`--json` 있음/없음으로 전체
  덤프)는 더 이상 지원하지 않는다.
- **`-L, --limit <N>`**: 결과 개수를 제한한다. 이슈 목록은 서버 페이지네이션(`size` 파라미터)을
  그대로 쓰고, PR/프로젝트 목록은 서버가 페이지네이션을 지원하지 않아 클라이언트 사이드 슬라이싱으로
  처리한다.
- **`--web`**: API 호출 대신 해당 리소스의 웹 페이지 URL을 브라우저로 연다(`view`/`list` 계열에
  존재).

## 명령어

### `yona auth`

| 명령 | 설명 |
|---|---|
| `yona auth login --server <URL> [--token <값>]` | 로그인 정보 저장 |
| `yona auth logout` | 로그인 정보 삭제 |
| `yona auth status` | 로그인 상태 표시(토큰은 마스킹) |

### `yona server` — 여러 서버 전환

`gh auth switch`에 대응하되, yona는 자체호스팅이라 회사/개인마다 완전히 다른 인스턴스를 오갈 일이
많아 별도 최상위 커맨드로 뒀다. `use`는 이미 로그인된 서버로 전환할 뿐 재로그인을 요구하지 않는다.

| 명령 | 설명 |
|---|---|
| `yona server list` | 로그인된 서버 목록(현재 서버는 `*` 표시) |
| `yona server use <서버 URL>` | 이미 로그인된 서버로 전환 |

### `yona browse` — 브라우저로 열기

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
| `yona issue edit <number> [-R ...] [--title] [--body]` | 제목/본문 수정 (생략한 필드는 기존 값 유지) |
| `yona issue comment <number> [-R ...] --body <내용>` | 댓글 작성 |
| `yona issue close <number> [-R ...]` | 닫기 |
| `yona issue reopen <number> [-R ...]` | 다시 열기 |
| `yona issue transfer <number> [-R ...] --to <owner/project>` | 다른 프로젝트로 이동 |
| `yona issue status [--json fields]` | 내가 담당/작성한 이슈 개수·목록(최소 버전 — mentioned/favorite/shared 필터는 서버가 아직 미지원) |

### `yona pr`

| 명령 | 설명 |
|---|---|
| `yona pr list [-R <owner/project>] [--state] [--author] [-L N] [--json fields] [--web]` | 목록 |
| `yona pr view <number> [-R ...] [--json fields] [--web]` | 조회 |
| `yona pr create -R <owner/project> --title <제목> --from <owner/project> --from-branch <브랜치> --to-branch <브랜치>` | 생성 (`--from`은 fork 프로젝트를 "owner/project" 형식으로 지정 — 숫자 ID를 미리 조회할 필요 없음) |
| `yona pr edit <number> [-R ...] [--title] [--body]` | 제목/본문 수정 (생략한 필드는 기존 값 유지) |
| `yona pr merge <number> [-R ...]` | 머지 |
| `yona pr close <number> [-R ...]` | 닫기 |
| `yona pr reopen <number> [-R ...]` | 다시 열기 |
| `yona pr diff <number> [-R ...] [--json fields]` | 변경된 파일 목록(pathA/pathB/changeType) |
| `yona pr comment <number> [-R ...] --body <내용>` | PR 전체에 댓글 작성 |
| `yona pr review <number> [-R ...]` | 본인을 리뷰어로 등록 (서버 API가 "리뷰어 지정"이 아니라 "자기등록" 방식) |
| `yona pr checkout <number> [-R ...]` | fromProject/fromBranch로 `git fetch` + `git checkout -B pr-<번호>` (서버 API 불필요) |

### `yona project`

| 명령 | 설명 |
|---|---|
| `yona project list <owner> [-L N] [--json fields]` | owner 아래 프로젝트 목록 |
| `yona project view <owner/project> [--json fields] [--web]` | 프로젝트 조회 |
| `yona project create <owner/name> [--overview] [--scope PUBLIC\|PRIVATE] [--vcs GIT]` | 생성 (세션 로그인/레거시 전권 토큰만 가능 — Fine-grained 스코프 토큰으로는 저장소 생성 불가) |
| `yona project fork <owner/project>` | 자신의 계정 아래로 fork |
| `yona project edit <owner/project> [--name] [--overview] [--scope] [--default-branch]` | 설정 수정 (생략한 필드는 기존 값 유지) |
| `yona project delete <owner/project> --yes` | 삭제 (되돌릴 수 없음, `--yes` 필수) |

### `yona label`

| 명령 | 설명 |
|---|---|
| `yona label list [-R <owner/project>] [--json fields]` | 목록 |
| `yona label create [-R ...] --name --color --category [--exclusive]` | 생성 |
| `yona label edit <id> [-R ...] --name --color --category-id` | 수정 |
| `yona label delete <id> [-R ...]` | 삭제 |

### `yona search`

| 명령 | 설명 |
|---|---|
| `yona search issues <query> [--page] [--size] [--json fields]` | 이슈 검색 |
| `yona search projects <query> [--page] [--size] [--json fields]` | 프로젝트 검색 |

`yona search prs`는 구현하지 않았다 — yona `SearchType` enum에 PR을 색인하는 값이 없어 서버 자체에
대응 기능이 없다(yona-wiki 계획 문서에 다음 라운드 이월로 기록됨).

### `yona org`

| 명령 | 설명 |
|---|---|
| `yona org list [--filter] [--page] [--json fields]` | 조직 목록 |
| `yona org view <name> [--json fields]` | 조직 조회(소속 프로젝트 포함) |

### `yona admin`

yona-wiki 계획 문서 Step9 조사 결과: 백업은 서버에 JSON에 가까운 API가 있어 완전히
연결했지만, 웹훅/권한 관리는 서버에 세션·폼 기반 레거시 컨트롤러만 있어 일부만 연결
가능했다. **목록 조회 두 개는 서버에 대응하는 JSON API가 없어 의도적으로 미구현
스텁**이다(자세한 내용은 `internal/api/admin.go` 상단 주석과 yona 저장소 계획 문서 참고).

| 명령 | 설명 | 상태 |
|---|---|---|
| `yona admin backup export [-o <파일>]` | `GET /site/export` 전체 백업 다운로드 (사이트 매니저 전용) | 구현됨 |
| `yona admin backup import <파일>` | `POST /site/import` 전체 복원 (기존 데이터 완전 교체) | 구현됨 |
| `yona admin webhook create --repo <owner/project> --url <URL> [--secret][--git-push][--type]` | 웹훅 생성 | 구현됨 |
| `yona admin webhook delete <id> --repo <owner/project>` | 웹훅 삭제 | 구현됨 |
| `yona admin webhook list --repo <owner/project>` | 웹훅 목록 | **미구현**(서버가 HTML만 반환) |
| `yona admin permission add <loginId> --repo <owner/project>` | 멤버 추가 | 구현됨 |
| `yona admin permission update-role <userId> <roleId> --repo <owner/project>` | 역할 변경 | 구현됨 |
| `yona admin permission remove <userId> --repo <owner/project>` | 멤버 제거 | 구현됨 |
| `yona admin permission list --repo <owner/project>` | 멤버/권한 목록 | **미구현**(서버에 엔드포인트 자체가 없음) |

### `yona api` (저수준 원시 호출)

`gh api`와 동일한 컨셉 — CLI가 아직 감싸지 않은 엔드포인트를 스크립팅/디버깅용으로 직접
호출한다.

```bash
yona api /api/v1/projects/acme
yona api -X POST -f title=hello -f body=world /api/v1/projects/acme/widgets/issues
yona api -X DELETE /api/v1/projects/acme/widgets/issues/1
echo '{"title":"raw body"}' | yona api -X POST --input - /api/v1/projects/acme/widgets/issues
```

### 기타

- `yona --version` — 버전 출력
- `yona completion [bash|zsh|fish|powershell]` — 쉘 자동완성 스크립트 생성 (Cobra 기본 제공)
- `yona config`, `yona alias` — 아직 구현하지 않음(nice-to-have, 낮은 우선순위로 다음 라운드 이후로 이월)

## 프로젝트 구조

```
.
├── main.go                  # 진입점
├── cmd/                      # Cobra 명령 트리 (auth/server/browse/issue/pr/project/label/search/org/admin/api)
├── internal/api/             # yona REST API HTTP 클라이언트
├── internal/gitutil/         # 로컬 git 연동 (--repo 자동감지, pr checkout, --web 브라우저 열기)
├── internal/weburl/          # 웹 UI 페이지 URL 계산 (--web/browse 공용)
└── internal/config/          # ~/.config/yona-cli/config.yml 로드/저장
```

## 테스트

```bash
go test ./...
```

실제 yona 서버 없이 `net/http/httptest`로 서버 응답을 목킹해 전부 검증한다(TDD).

## 알려진 한계 / 다음 단계

- `yona auth login → 이슈 생성 → PR 목록 조회` 골든 패스의 실서버 수동 검증은 다음 라운드로
  남겨뒀다(이번 라운드도 단위/통합 테스트로 HTTP 클라이언트 로직만 검증).
- 웹훅/권한 목록 조회는 서버에 JSON API가 없어 미구현이다(위 표 참고). 서버 쪽에 API를
  추가하는 것은 이 CLI 프로젝트의 범위 밖이다.
- `pr diff`가 그대로 노출하는 서버 응답(`List<FileDiff>`)은 JGit 내부 타입(`RawText`/
  `EditList`)을 그대로 직렬화하는 구조라 일반적인 Jackson 빈 컨벤션에 맞지 않는다 — CLI는
  `pathA`/`pathB`/`changeType`만 안전하게 쓰고 나머지는 `--json`으로만 노출한다(서버 쪽 개선은
  이 CLI 작업 범위 밖).
- `yona search prs`, `yona issue status`의 mentioned/favorite/shared 필터·페이지네이션 확장은
  서버(yona) 쪽 기능 자체가 아직 없어 이월됐다(yona-wiki 계획 문서 참고).
- `search`/`organizations`/`user` 전역 엔드포인트와 `project create`는 저장소 단위 스코프
  모델과 맞지 않아 Fine-grained 스코프 토큰으로는 인증되지 않는다(세션 로그인/레거시 전권
  토큰만 가능 — yona-wiki 계획 문서의 스코프 인가 갭 기록 참고).
- `yona config`, `yona alias`는 낮은 우선순위(nice-to-have)로 아직 구현하지 않았다.
- 배포(goreleaser, Step 11)는 이번 라운드 범위 밖이다.
- SSH 인증(`yona auth login --with-ssh`), `yona runner`/`yona workflow`, `yona mcp serve`는
  각각 별도 계획([[p3-03-ssh-gpg]], [[p3-05-ci-actions-runner]], [[p3-07-mcp-server]])에서
  다룬다.

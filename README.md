# yona-cli

`yona-cli`는 [yona](https://github.com/yona-projects/yona) 서버(REST API)를 감싸는 커맨드라인 도구다.
GitHub CLI(`gh`)와 동일한 컨셉/스택(Go + Cobra)으로 만들었다 — 설치 직후 바로 실행돼야 하므로
JVM 콜드스타트가 있는 Kotlin은 제외하고 Go를 채택했다(자세한 배경은 yona 저장소의
`docs/yona-wiki/plans/p3-02-cli-and-rest-api.md` 참고).

## 설치

```bash
curl -fsSL https://raw.githubusercontent.com/yona-projects/yona-cli/main/install.sh | sh
```

Linux/macOS(amd64/arm64) 바이너리를 [GitHub Releases](https://github.com/yona-projects/yona-cli/releases)에서
받아 `~/.local/bin`에 설치한다(`YONA_INSTALL_DIR`로 경로 변경, `YONA_VERSION`으로 특정 버전 고정
가능). Windows는 Releases 페이지에서 `.zip`을 직접 받으면 된다. 릴리즈는 `scripts/release.sh`로
만든다(크로스컴파일 + `gh release create` — goreleaser 같은 별도 도구 없이 `gh`만 쓴다).

소스에서 직접 빌드하려면:

```bash
go build -o bin/yona .
```

(Homebrew/Scoop/`.deb`/`.rpm` 패키지 배포는 아직 범위 밖이다)

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
| `yona issue status [--state] [--filter] [--page] [--json fields]` | 담당/작성/댓글단/멘션된/즐겨찾기한/공유받은 이슈 현황(6개 섹션) |

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
| `yona pr review <number> [-R ...]` | 본인을 리뷰어로 등록(`--remove`시 등록 취소, 서버 API가 "리뷰어 지정"이 아니라 "자기등록" 방식) — `--approve`/`--request-changes`/`--comment`(+`--body`) 플래그를 주면 자기등록과 별개로 Approve/Request changes/Comment 판정을 제출한다 |
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

### `yona tag`

| 명령 | 설명 |
|---|---|
| `yona tag list [-R <owner/project>] [--json fields]` | 목록(lightweight/annotated 구분, annotated면 message/tagger 포함) |
| `yona tag create <name> [-R ...] [--target] [--message]` | 생성 — `--target` 생략 시 기본 브랜치 HEAD, `--message`를 주면 annotated 태그(태거는 API 토큰 소유자), 생략하면 lightweight 태그 |
| `yona tag delete <name> [-R ...]` | 삭제 (브랜치 삭제와 동일하게 매니저/조직관리자 권한 필요) |

`gh` CLI에는 독립된 최상위 `gh tag`가 없다(태그는 `gh release`에 종속되거나 순수 `git tag`/`git push
--tags`로 다룬다) — yona는 릴리즈 개념이 없어 이 명령이 태그 관리의 유일한 CLI 경로다.

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

### `yona config` — 로컬 설정값 관리

`gh config get/set/list`에 대응한다. 알 수 없는 키는 오류로 처리한다(오타를 조용히 저장하지
않기 위함).

| 명령 | 설명 |
|---|---|
| `yona config get <key>` | 설정값 조회 |
| `yona config set <key> <value>` | 설정값 저장 |
| `yona config list` | 지원 키와 현재 값 목록 |

현재 지원하는 키: `browser`(`--web`/`browse` 계열이 열 브라우저 명령 — `BROWSER` 환경변수가
이 설정보다 우선한다).

### `yona alias` — 명령 별칭

`gh alias set/list/delete`에 대응한다. `gh`와 달리 `!`로 시작하는 셸 명령 별칭은 지원하지
않는다 — 저장된 문자열을 셸로 그대로 실행하면 config.yml을 편집할 수 있는 사람이 임의 명령을
실행시킬 수 있는 경로가 생기므로, 다른 yona 하위 명령으로만 확장되는 별칭만 허용한다.

| 명령 | 설명 |
|---|---|
| `yona alias set <name> <expansion>` | 별칭 등록 (예: `yona alias set pv "pr view"`) |
| `yona alias list` | 등록된 별칭 목록 |
| `yona alias delete <name>` | 별칭 삭제 |

기존 최상위 명령과 이름이 겹치는 별칭은 등록할 수 없다(내장 명령이 항상 우선).

### 기타

- `yona --version` — 버전 출력
- `yona completion [bash|zsh|fish|powershell]` — 쉘 자동완성 스크립트 생성 (Cobra 기본 제공)

## 프로젝트 구조

```
.
├── main.go                  # 진입점
├── install.sh                # 원커맨드 설치 스크립트 (curl -fsSL ... | sh)
├── scripts/release.sh        # 크로스컴파일 + gh release create로 릴리즈 발행
├── cmd/                      # Cobra 명령 트리 (auth/server/browse/issue/pr/project/label/tag/search/org/admin/api/config/alias)
├── internal/api/             # yona REST API HTTP 클라이언트
├── internal/gitutil/         # 로컬 git 연동 (--repo 자동감지, pr checkout, --web 브라우저 열기)
├── internal/weburl/          # 웹 UI 페이지 URL 계산 (--web/browse 공용)
└── internal/config/          # ~/.config/yona-cli/config.yml 로드/저장 (호스트/설정값/별칭)
```

## 테스트

```bash
go test ./...
```

실제 yona 서버 없이 `net/http/httptest`로 서버 응답을 목킹해 전부 검증한다(TDD).

## 알려진 한계 / 다음 단계

해결된 항목:

- `yona auth login → 이슈 생성 → PR 목록 조회` 골든 패스는 실서버+실CLI로 반복 검증됐다(그
  과정에서 발견된 실버그도 함께 수정됨).
- 웹훅/권한 목록 조회(`yona admin webhook list`, `yona admin permission list`)는 서버가
  지원하는 목록 API에 CLI가 배선돼 더 이상 스텁이 아니다.
- `pr diff`는 서버가 JGit 내부 타입 대신 `patch`(unified diff) 필드를 내려주도록 개선돼,
  CLI도 실제 diff 내용을 출력한다.
- `yona search prs`는 서버 쪽에 검색 기능이 추가된 뒤 CLI에 배선 완료됐다.
- `yona issue status`가 서버의 commented/mentioned/favorite/shared 4개 섹션과
  state/filter/pageNum 파라미터를 쓰도록 CLI를 배선했다(서버는 이미 지원 중이었고 CLI만
  뒤처져 있었다).
- `yona config get/set/list`, `yona alias set/list/delete`를 구현했다(`browser` 설정값 하나로
  시작 — `--web`/`browse` 계열이 열 브라우저를 오버라이드한다. 별칭은 다른 yona 하위 명령으로만
  확장되고, `gh`의 `!` 셸 명령 별칭처럼 임의 셸 실행을 허용하지는 않는다).
- GitHub Releases 배포(`install.sh` + `scripts/release.sh` + `gh release create`)를 구현했다
  — goreleaser 같은 별도 도구 없이 `gh`만으로 크로스컴파일·업로드까지 처리한다.

아직 남은 항목:

- `search`/`organizations` 전역 엔드포인트는 최근 Fine-grained PAT 인증이 추가됐지만,
  `project create`는 여전히 세션 로그인/레거시 전권 토큰만 가능하다(저장소 단위 스코프
  모델과 안 맞는 "저장소를 새로 만드는" 요청이라 — 버그가 아니라 서버 스코프 모델의 구조적
  제약으로 계획 문서에 의도된 설계로 기록돼 있다).
- SSH 인증: CLI 내부에 `ssh-auth`/`ssh-shell` 서브커맨드를 시도했으나, 실제 배포 경로는
  서버 내부 API를 직접 호출하는 bash 스크립트 방식임이 확인돼 죽은 코드로 판단하고 제거했다
  (서버 쪽 SSH 지원 자체는 완료됨 — [[p3-03-ssh-gpg]]). CLI 쪽에 추가로 할 일은 없다.
- `yona runner`/`yona workflow`는 아직 착수하지 않았다([[p3-05-ci-actions-runner]]).
- `yona mcp serve`는 CLI 명령으로 만드는 안이 채택되지 않았다 — OAuth 인가 서버 역할 때문에
  yona 자신의 로그인 스택이 필요해, MCP 서버 자체를 Kotlin/Spring 서버 안에 내장하는 쪽으로
  확정·구현 완료됐다([[p3-07-mcp-server]]). CLI 쪽에 추가로 할 일은 없다.
- Homebrew/Scoop/`.deb`/`.rpm` 패키지 배포는 각각 별도 tap/bucket 저장소가 필요해 아직
  범위 밖이다.

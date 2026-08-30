# yona-cli

`yona`는 [yuna](https://github.com/search5/yuna) 서버(REST API)를 감싸는 커맨드라인 도구다.
GitHub CLI(`gh`)와 동일한 컨셉/스택(Go + Cobra)으로 만들었다 — 설치 직후 바로 실행돼야 하므로
JVM 콜드스타트가 있는 Kotlin은 제외하고 Go를 채택했다(자세한 배경은 yuna 저장소의
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

# 3. 이슈/PR/프로젝트 조회
yona project list acme
yona project view acme/widgets
yona issue list --repo acme/widgets
yona issue view 42 --repo acme/widgets
yona pr list --repo acme/widgets

# 4. 로그아웃
yona auth logout
```

### 인증 토큰의 기본 스코프

`yona auth login`은 **본인 계정이 가진 전체 권한**(전체 저장소 + 모든 스코프 그룹 write)을
전제로 한다 — 웹 세션 로그인과 동등한 권한이라는 뜻이다. yuna 서버 쪽에 OAuth 유사 로그인
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

## 명령어

### `yona auth`

| 명령 | 설명 |
|---|---|
| `yona auth login --server <URL> [--token <값>]` | 로그인 정보 저장 |
| `yona auth logout` | 로그인 정보 삭제 |
| `yona auth status` | 로그인 상태 표시(토큰은 마스킹) |

### `yona issue`

| 명령 | 설명 |
|---|---|
| `yona issue list --repo <owner/project> [--state STATE] [--json]` | 목록 |
| `yona issue view <number> --repo <owner/project> [--json]` | 조회 |
| `yona issue create --repo <owner/project> --title <제목> [--body <본문>] [--draft]` | 생성 |
| `yona issue comment <number> --repo <owner/project> --body <내용>` | 댓글 작성 |
| `yona issue close <number> --repo <owner/project>` | 닫기 |

### `yona pr`

| 명령 | 설명 |
|---|---|
| `yona pr list --repo <owner/project> [--state STATE] [--json]` | 목록 |
| `yona pr view <number> --repo <owner/project> [--json]` | 조회 |
| `yona pr create --repo <owner/project> --title <제목> --from-project-id <ID> --from-branch <브랜치> --to-branch <브랜치>` | 생성 |
| `yona pr merge <number> --repo <owner/project>` | 머지 |
| `yona pr review <number> --repo <owner/project>` | 본인을 리뷰어로 등록 (서버 API가 "리뷰어 지정"이 아니라 "자기등록" 방식) |

### `yona project`

| 명령 | 설명 |
|---|---|
| `yona project list <owner> [--json]` | owner 아래 프로젝트 목록 |
| `yona project view <owner/project> [--json]` | 프로젝트 조회 |

### `yona admin`

yona-wiki 계획 문서 Step9 조사 결과: 백업은 서버에 JSON에 가까운 API가 있어 완전히
연결했지만, 웹훅/권한 관리는 서버에 세션·폼 기반 레거시 컨트롤러만 있어 일부만 연결
가능했다. **목록 조회 두 개는 서버에 대응하는 JSON API가 없어 의도적으로 미구현
스텁**이다(자세한 내용은 `internal/api/admin.go` 상단 주석과 yuna 저장소 계획 문서 참고).

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

## 프로젝트 구조

```
.
├── main.go                  # 진입점
├── cmd/                      # Cobra 명령 트리 (auth/issue/pr/project/admin/api)
├── internal/api/             # yuna REST API HTTP 클라이언트
└── internal/config/          # ~/.config/yona-cli/config.yml 로드/저장
```

## 테스트

```bash
go test ./...
```

실제 yuna 서버 없이 `net/http/httptest`로 서버 응답을 목킹해 전부 검증한다(TDD).

## 알려진 한계 / 다음 단계

- `yona auth login → 이슈 생성 → PR 목록 조회` 골든 패스의 실서버 수동 검증은 다음 라운드로
  남겨뒀다(이번 라운드는 단위/통합 테스트로 HTTP 클라이언트 로직만 검증).
- 웹훅/권한 목록 조회는 서버에 JSON API가 없어 미구현이다(위 표 참고). 서버 쪽에 API를
  추가하는 것은 이 CLI 프로젝트의 범위 밖이다.
- 배포(goreleaser, Step 11)는 이번 라운드 범위 밖이다.
- SSH 인증(`yona auth login --with-ssh`), `yona runner`/`yona workflow`, `yona mcp serve`는
  각각 별도 계획([[p3-03-ssh-gpg]], [[p3-05-ci-actions-runner]], [[p3-07-mcp-server]])에서
  다룬다.

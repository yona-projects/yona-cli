#!/usr/bin/env bash
# yona-cli 릴리즈 아카이브를 크로스컴파일해 dist/에 만들고, gh(GitHub CLI)로 GitHub Release를
# 생성/업로드한다. goreleaser 없이 "gh를 최대한 활용한다"는 프로젝트 방향에 맞춘 최소 구현이다
# — cmd/root.go의 Version 변수를 ldflags로 주입하는 방식은 goreleaser와 동일하다.
#
# 사용법: scripts/release.sh <version>   (예: scripts/release.sh v0.1.0)
set -euo pipefail

if [ $# -ne 1 ]; then
  echo "사용법: $0 <version> (예: $0 v0.1.0)" >&2
  exit 1
fi

TAG="$1"
VERSION="${TAG#v}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
MODULE="github.com/yona-projects/yona-cli"

sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

echo "==> go test ./..."
(cd "$ROOT_DIR" && go test ./...)

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

PLATFORMS=(
  "linux amd64"
  "linux arm64"
  "darwin amd64"
  "darwin arm64"
  "windows amd64"
)

for platform in "${PLATFORMS[@]}"; do
  read -r goos goarch <<<"$platform"
  binary="yona"
  [ "$goos" = "windows" ] && binary="yona.exe"
  archive_base="yona_${VERSION}_${goos}_${goarch}"

  workdir="$(mktemp -d)"
  echo "==> building ${goos}/${goarch}"
  (cd "$ROOT_DIR" && GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build \
    -ldflags "-s -w -X ${MODULE}/cmd.Version=${VERSION}" \
    -o "$workdir/$binary" .)
  cp "$ROOT_DIR/README.md" "$workdir/"

  if [ "$goos" = "windows" ]; then
    (cd "$workdir" && zip -q "$DIST_DIR/${archive_base}.zip" "$binary" README.md)
  else
    tar -czf "$DIST_DIR/${archive_base}.tar.gz" -C "$workdir" "$binary" README.md
  fi
  rm -rf "$workdir"
done

echo "==> writing checksums.txt"
(
  cd "$DIST_DIR"
  : > checksums.txt
  for f in yona_*; do
    echo "$(sha256_of "$f")  $f" >> checksums.txt
  done
)

echo "==> dist 아카이브:"
ls -la "$DIST_DIR"

echo "==> git tag ${TAG}"
(cd "$ROOT_DIR" && git tag -a "$TAG" -m "yona-cli ${TAG}")
(cd "$ROOT_DIR" && git push origin "$TAG")

echo "==> gh release create ${TAG}"
(cd "$ROOT_DIR" && gh release create "$TAG" "$DIST_DIR"/* \
  --title "yona-cli ${TAG}" \
  --generate-notes)

echo "==> 완료: https://github.com/yona-projects/yona-cli/releases/tag/${TAG}"

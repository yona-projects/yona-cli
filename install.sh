#!/usr/bin/env sh
# yona-cli 원커맨드 설치 스크립트 (uv/rustup 스타일):
#   curl -fsSL https://raw.githubusercontent.com/yona-projects/yona-cli/main/install.sh | sh
#
# GitHub Releases(scripts/release.sh가 올린 아카이브)에서 OS/아키텍처에 맞는 바이너리를 받아
# 체크섬을 검증하고 설치한다. YONA_VERSION(기본 latest)과 YONA_INSTALL_DIR(기본
# $HOME/.local/bin)로 동작을 바꿀 수 있다.
set -eu

REPO="yona-projects/yona-cli"
INSTALL_DIR="${YONA_INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${YONA_VERSION:-latest}"

detect_os() {
  case "$(uname -s)" in
    Linux) echo linux ;;
    Darwin) echo darwin ;;
    *) echo "지원하지 않는 OS입니다: $(uname -s)" >&2; exit 1 ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo amd64 ;;
    arm64|aarch64) echo arm64 ;;
    *) echo "지원하지 않는 아키텍처입니다: $(uname -m)" >&2; exit 1 ;;
  esac
}

sha256_of() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

os="$(detect_os)"
arch="$(detect_arch)"

if [ "$VERSION" = "latest" ]; then
  tag="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | head -1 | cut -d '"' -f4)"
else
  tag="$VERSION"
fi
if [ -z "${tag:-}" ]; then
  echo "릴리즈 태그를 확인할 수 없습니다(네트워크 또는 API 제한 문제일 수 있습니다)." >&2
  exit 1
fi

ver="${tag#v}"
archive="yona_${ver}_${os}_${arch}.tar.gz"
base_url="https://github.com/${REPO}/releases/download/${tag}"

workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT

echo "yona-cli ${tag} (${os}/${arch}) 다운로드 중..."
curl -fsSL "${base_url}/${archive}" -o "$workdir/$archive"

if curl -fsSL "${base_url}/checksums.txt" -o "$workdir/checksums.txt" 2>/dev/null; then
  expected="$(grep " ${archive}\$" "$workdir/checksums.txt" | awk '{print $1}')"
  if [ -n "${expected:-}" ]; then
    actual="$(sha256_of "$workdir/$archive")"
    if [ "$expected" != "$actual" ]; then
      echo "체크섬이 일치하지 않습니다 (기대: $expected, 실제: $actual)" >&2
      exit 1
    fi
    echo "체크섬 확인 완료"
  fi
fi

tar -xzf "$workdir/$archive" -C "$workdir"
mkdir -p "$INSTALL_DIR"
install -m 0755 "$workdir/yona" "$INSTALL_DIR/yona"

echo "설치 완료: $INSTALL_DIR/yona"
"$INSTALL_DIR/yona" --version

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo ""
    echo "경고: $INSTALL_DIR 가 PATH에 없습니다. 셸 설정 파일에 다음을 추가하세요:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    ;;
esac

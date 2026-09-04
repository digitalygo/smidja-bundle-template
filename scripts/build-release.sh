#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

validate_version() {
  local version="$1"
  local pattern='^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$'
  [[ "$version" =~ $pattern ]]
}

if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
  return 0
fi

cd "$ROOT"

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <version> [outdir]" >&2
  exit 2
fi

version="$1"
outdir="${2:-dist}"

if ! validate_version "$version"; then
  echo "build-release: version must look like v1.2.3 or v1.2.3-beta.1, got '$version'" >&2
  exit 2
fi

module="$(go list -m)"
commit="$(git rev-parse HEAD 2>/dev/null || true)"
if [[ ! "$commit" =~ ^[0-9a-f]{7,64}$ ]]; then
  commit="none"
fi

targets=(
  linux amd64
  linux arm64
  darwin amd64
  darwin arm64
)

mkdir -p "$outdir"
rm -f "$outdir"/smidja-* "$outdir/checksums.txt"

ldflags="-s -w \
  -X main.bundleOrigin=${module} \
  -X main.bundleVersion=${version} \
  -X main.bundleCommit=${commit}"

for ((i = 0; i < ${#targets[@]}; i += 2)); do
  goos="${targets[i]}"
  goarch="${targets[i + 1]}"
  name="smidja-${goos}-${goarch}"
  echo "building ${name}"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "$ldflags" \
    -o "$outdir/$name" \
    .
done

if command -v sha256sum >/dev/null 2>&1; then
  sumcmd=(sha256sum)
else
  sumcmd=(shasum -a 256)
fi

(
  cd "$outdir"
  "${sumcmd[@]}" smidja-linux-amd64 smidja-linux-arm64 smidja-darwin-amd64 smidja-darwin-arm64 | LC_ALL=C sort -k 2 > checksums.txt
  "${sumcmd[@]}" -c checksums.txt
)

echo "release artifacts in $outdir:"
ls -la "$outdir"

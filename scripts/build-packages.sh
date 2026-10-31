set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
output="${1:-$root/dist}"
base="${2:-https://publisher.example/packages/example.coding/1.0.0}"
mkdir -p "$output"
output="$(cd "$output" && pwd)"
staging="$(mktemp -d)"
trap 'rm -rf "$staging"' EXIT
cd "$root"
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
  rm -rf "$staging/payload"
  cp -R package "$staging/payload"
  mkdir -p "$staging/payload/bin"
  GOOS="${target%/*}" GOARCH="${target#*/}" CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -buildid=" -o "$staging/payload/bin/helper" ./cmd/helper
  chmod 755 "$staging/payload/bin/helper"
  python3 scripts/package_builder.py "$staging/payload" "$output" "$target" "$base"
done

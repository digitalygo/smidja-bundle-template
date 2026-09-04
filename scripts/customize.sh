#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

TEMPLATE_MODULE="$(go list -m)"
TEMPLATE_ORIGIN="$TEMPLATE_MODULE"

usage() {
  echo "usage: $0 github.com/owner/repo" >&2
  echo "  rewrites the go module path, bundle origin, and go imports from" >&2
  echo "  $TEMPLATE_ORIGIN to the given origin, then builds to verify" >&2
  exit 2
}

if [[ $# -ne 1 ]]; then
  usage
fi

TARGET_ORIGIN="$1"

if [[ "$TARGET_ORIGIN" == "$TEMPLATE_ORIGIN" ]]; then
  echo "customize: $TARGET_ORIGIN is the template origin, nothing to customize" >&2
  usage
fi

if [[ ! "$TARGET_ORIGIN" =~ ^github\.com/[A-Za-z0-9][A-Za-z0-9-]{0,38}/[A-Za-z0-9._-]+$ ]]; then
  echo "customize: origin '$TARGET_ORIGIN' must look like github.com/owner/repo" >&2
  usage
fi

if [[ -n "$(git -C "$ROOT" status --porcelain 2>/dev/null)" ]]; then
  echo "customize: working tree is dirty, commit or stash before customizing" >&2
  exit 1
fi

go mod edit -module "$TARGET_ORIGIN"

match_count=$(grep -rl --include='*.go' -F "$TEMPLATE_ORIGIN" . | wc -l)
if [[ "$match_count" -eq 0 ]]; then
  echo "customize: no go files reference $TEMPLATE_ORIGIN, nothing to rewrite" >&2
  exit 1
fi

grep -rl --include='*.go' -F "$TEMPLATE_ORIGIN" . | while read -r file; do
  sed -i.bak "s|$TEMPLATE_ORIGIN|$TARGET_ORIGIN|g" "$file"
  rm -f "$file.bak"
  echo "customize: rewrote $file"
done

go build ./...

echo "customize: bundle renamed to $TARGET_ORIGIN"
echo "customize: next steps:"
echo "  1. adjust the prose in README.md and AGENTS.md to the new bundle name"
echo "  2. commit the rename on your first commit"
echo "  3. push a v* tag to publish the first release with assets smidja-<os>-<arch>"

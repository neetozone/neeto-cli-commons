#!/bin/bash
set -euo pipefail

COMMONS_VERSION="${1:-}"
COMMONS_REPO="github.com/neetozone/neeto-cli-commons"
ORG="neetozone"

if [ -z "$COMMONS_VERSION" ]; then
  echo "usage: bump.sh <commons version, e.g. v0.2.0>" >&2
  exit 1
fi

for tool in git go gh; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "Required tool not found on PATH: $tool" >&2
    exit 1
  fi
done

export GOPRIVATE="${ORG_GOPRIVATE:-github.com/neetozone/*}"
if [ -n "${GITHUB_TOKEN:-}" ]; then
  git config --global url."https://x-access-token:${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"
fi

PRODUCTS=$(grep -v '^[[:space:]]*$' products.txt | grep -v '^#')
BRANCH="bump-neeto-cli-commons-${COMMONS_VERSION}"
WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT INT TERM

FAILED=""
OPENED=""

for repo in $PRODUCTS; do
  echo "=== $repo ==="
  target="$WORKDIR/$repo"

  if ! git clone --quiet --depth 1 "https://github.com/${ORG}/${repo}.git" "$target"; then
    echo "$repo: clone failed" >&2
    FAILED="$FAILED $repo"
    continue
  fi

  set +e
  (
    cd "$target"

    git checkout -q -b "$BRANCH"

    go mod edit -require="${COMMONS_REPO}@${COMMONS_VERSION}"
    go mod tidy

    if grep -q '^commons_version:' .neeto-cli.yml; then
      sed -i.bak "s|^commons_version:.*|commons_version: ${COMMONS_VERSION}|" .neeto-cli.yml && rm -f .neeto-cli.yml.bak
    else
      printf 'commons_version: %s\n' "${COMMONS_VERSION}" >> .neeto-cli.yml
    fi

    go run "${COMMONS_REPO}/cmd/neeto-cli-sync"

    gofmt -l . | tee /tmp/bump_fmt && test ! -s /tmp/bump_fmt
    go build ./...
    go vet ./...
    go test ./...
    go run "${COMMONS_REPO}/cmd/neeto-cli-sync" --check

    if git diff --quiet; then
      echo "$repo: already on ${COMMONS_VERSION}, nothing to do"
      exit 3
    fi

    git add -A
    git -c commit.gpgsign=false commit -q -m "Bumped neeto-cli-commons to ${COMMONS_VERSION}"
    git push -q origin "${BRANCH}:${BRANCH}"

    gh pr create \
      --repo "${ORG}/${repo}" \
      --base main \
      --head "$BRANCH" \
      --title "Bumped neeto-cli-commons to ${COMMONS_VERSION}" \
      --label patch \
      --body "Picks up neeto-cli-commons ${COMMONS_VERSION}. Generated files were re-synced and the build, vet and test suites pass."
  )
  status=$?
  set -e

  case $status in
    0) OPENED="$OPENED $repo" ;;
    3) ;;
    *) echo "$repo: bump failed" >&2; FAILED="$FAILED $repo" ;;
  esac
done

echo
echo "Opened PRs for:${OPENED:- none}"
if [ -n "$FAILED" ]; then
  echo "Failed:${FAILED}" >&2
  exit 1
fi

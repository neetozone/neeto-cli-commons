#!/bin/bash
set -euo pipefail

COMMONS_VERSION="${1:-}"
COMMONS_REPO="github.com/neetozone/neeto-cli-commons"
ORG="neetozone"

if [ -z "$COMMONS_VERSION" ]; then
  echo "usage: bump.sh <commons version, e.g. v0.3.0>" >&2
  exit 1
fi

for tool in git go gh; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "Required tool not found on PATH: $tool" >&2
    exit 1
  fi
done

export GOPRIVATE="github.com/neetozone/*"
if [ -n "${GITHUB_TOKEN:-}" ]; then
  git config --global url."https://x-access-token:${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"
fi

PRODUCTS=$(grep -vE '^[[:space:]]*(#|$)' products.txt)
BRANCH="bump-neeto-cli-commons-${COMMONS_VERSION}"
WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT INT TERM

FAILED=""
OPENED=""
SKIPPED=""
UNPORTED=""

for repo in $PRODUCTS; do
  echo "=== $repo ==="
  target="$WORKDIR/$repo"

  if ! git clone --quiet "https://github.com/${ORG}/${repo}.git" "$target"; then
    echo "$repo: clone failed" >&2
    FAILED="$FAILED $repo"
    continue
  fi

  set +e
  (
    set -euo pipefail
    cd "$target"

    git config user.name "NeetoBot"
    git config user.email "bot@neeto.com"
    git config core.hooksPath /dev/null
    git checkout -q -b "$BRANCH"

    if [ ! -f .neeto-cli.yml ]; then
      echo "$repo: no .neeto-cli.yml — this repo has not been ported to neeto-cli-commons yet" >&2
      exit 4
    fi
    if ! grep -q "${COMMONS_REPO}" go.mod; then
      echo "$repo: go.mod does not require ${COMMONS_REPO} — not ported yet" >&2
      exit 4
    fi

    go mod edit -require="${COMMONS_REPO}@${COMMONS_VERSION}"
    go mod tidy

    if grep -q '^commons_version:' .neeto-cli.yml; then
      sed -i.bak "s|^commons_version:.*|commons_version: ${COMMONS_VERSION}|" .neeto-cli.yml
      rm -f .neeto-cli.yml.bak
    else
      printf 'commons_version: %s\n' "${COMMONS_VERSION}" >> .neeto-cli.yml
    fi

    if ! grep -q "^	${COMMONS_REPO} ${COMMONS_VERSION}\$" go.mod; then
      echo "$repo: go mod tidy dropped ${COMMONS_VERSION} from go.mod" >&2
      exit 5
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
    git commit -q -m "Bumped neeto-cli-commons to ${COMMONS_VERSION}"
    git push -q --force origin "${BRANCH}:${BRANCH}"

    existing=$(gh pr list --repo "${ORG}/${repo}" --head "$BRANCH" --state open --json number --jq '.[0].number')
    if [ -n "$existing" ]; then
      echo "$repo: PR #${existing} already open for ${BRANCH}"
      exit 0
    fi

    gh pr create \
      --repo "${ORG}/${repo}" \
      --base main \
      --head "$BRANCH" \
      --title "Bumped neeto-cli-commons to ${COMMONS_VERSION}" \
      --label patch \
      --body "Picks up neeto-cli-commons ${COMMONS_VERSION}. The generated files were re-synced and the build, vet, gofmt and test suites all pass. Merging this with the patch label cuts the next release of this CLI."
  )
  status=$?
  set -e

  case $status in
    0) OPENED="$OPENED $repo" ;;
    3) SKIPPED="$SKIPPED $repo" ;;
    4) UNPORTED="$UNPORTED $repo" ;;
    *) echo "$repo: bump failed" >&2; FAILED="$FAILED $repo" ;;
  esac
done

echo
echo "Opened:${OPENED:- none}"
echo "Already current:${SKIPPED:- none}"
echo "Not ported yet:${UNPORTED:- none}"
if [ -n "$FAILED" ] || [ -n "$UNPORTED" ]; then
  echo "Failed:${FAILED:- none}" >&2
  exit 1
fi

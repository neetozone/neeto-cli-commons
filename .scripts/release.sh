#!/bin/bash
set -euo pipefail

LAST_COMMIT_MSG=$(git log -1 --format="%s")
if echo "$LAST_COMMIT_MSG" | grep -q "^Bump version to "; then
  echo "Last commit is an automated version bump. Skipping release to prevent infinite loop."
  exit 0
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "gh is not on PATH; installing it."
  curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
  sudo chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    | sudo tee /etc/apt/sources.list.d/github-cli.list >/dev/null
  sudo apt-get update -qq
  sudo apt-get install -y -qq gh
fi

for tool in git go gh goreleaser; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "Required tool not found on PATH: $tool" >&2
    exit 1
  fi
done

PR_NUMBER=$(gh pr list --state merged --base main --limit 50 --json number,mergedAt --jq "sort_by(.mergedAt) | last | .number")
if [ -z "$PR_NUMBER" ]; then
  echo "No merged PR found. Skipping release."
  exit 0
fi

PR_LABELS=$(gh pr view "$PR_NUMBER" --json labels --jq ".labels[].name" | tr "\n" " ")
echo "Last merged PR: #${PR_NUMBER} with labels: ${PR_LABELS}"

VERSION_LABEL=""
for label in major minor patch; do
  if echo "$PR_LABELS" | grep -qw "$label"; then
    VERSION_LABEL="$label"
    break
  fi
done

if [ -z "$VERSION_LABEL" ]; then
  echo "No version label found. Skipping release."
  exit 0
fi

git fetch --quiet --tags origin
FILE_VERSION=$(tr -d '[:space:]' < VERSION)
LATEST_TAG=$(git tag --list 'v[0-9]*' --sort=-v:refname | head -n 1)
CURRENT_VERSION=$(printf '%s\n%s\n' "${FILE_VERSION#v}" "${LATEST_TAG#v}" | sort -V | tail -n 1)
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

case "$VERSION_LABEL" in
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  patch) PATCH=$((PATCH + 1)) ;;
esac

VERSION="${MAJOR}.${MINOR}.${PATCH}"
echo "Releasing v${VERSION} (bumped from ${CURRENT_VERSION} via the ${VERSION_LABEL} label)"

git config core.hooksPath /dev/null

git config user.name "NeetoBot"
git config user.email "bot@neeto.com"

git fetch origin main
git checkout main
git pull --ff-only origin main

if git rev-parse "v${VERSION}" >/dev/null 2>&1; then
  echo "Tag v${VERSION} already exists. Skipping tag creation."
else
  git tag -a "v${VERSION}" -m "Release v${VERSION}"
  git push origin "v${VERSION}"
  echo "Pushed tag v${VERSION}"
fi

BUMP_BRANCH="bump-version-to-${VERSION}"
if [ -n "$(gh pr list --head "$BUMP_BRANCH" --state open --json number --jq '.[].number')" ]; then
  echo "A pull request bumping VERSION to ${VERSION} is already open."
else
  git checkout -B "$BUMP_BRANCH"
  echo "$VERSION" > VERSION
  git add VERSION
  git commit -m "Bump version to $VERSION"
  git push --force origin "$BUMP_BRANCH"
  gh pr create --base main --head "$BUMP_BRANCH" --label mergepr \
    --title "Bump version to $VERSION" \
    --body "Records v${VERSION} in VERSION after the release. main only takes changes through pull requests, so the release pipeline raises the bump here and mergepr merges it once CI passes."
  git checkout main
fi

RELEASE_STATUS=0

echo "Publishing the neeto-cli-gen binaries for v${VERSION}..."
export GORELEASER_CURRENT_TAG="v${VERSION}"
if ! goreleaser release --clean; then
  echo "goreleaser failed. The tag is already pushed and the version bump pull request is open, so any product roll-out below still runs; re-run goreleaser against v${VERSION} once the cause is fixed." >&2
  RELEASE_STATUS=1
fi

if [ "$VERSION_LABEL" = "major" ]; then
  echo "Skipping the product roll-out because this is a major release."
  echo "Go requires a /vN suffix on the module path from v2 onwards, so the products cannot"
  echo "resolve v${VERSION} until github.com/neetozone/neeto-cli-commons becomes"
  echo "github.com/neetozone/neeto-cli-commons/v${MAJOR} and every import in every product is"
  echo "updated to match. Do that first, then run .scripts/bump.sh v${VERSION} by hand."
else
  echo "Opening the version-bump pull requests on every product CLI..."
  bash .scripts/bump.sh "v${VERSION}"
fi

exit "$RELEASE_STATUS"

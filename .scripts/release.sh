#!/bin/bash
set -euo pipefail

LAST_COMMIT_MSG=$(git log -1 --format="%s")
if echo "$LAST_COMMIT_MSG" | grep -q "^Bump version to "; then
  echo "Last commit is an automated version bump. Skipping release to prevent infinite loop."
  exit 0
fi

for tool in git go gh; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "Required tool not found on PATH: $tool" >&2
    exit 1
  fi
done

PR_NUMBER=$(gh pr list --state merged --base main --limit 1 --json number --jq ".[0].number")
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

CURRENT_VERSION=$(tr -d '[:space:]' < VERSION)
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

case "$VERSION_LABEL" in
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  patch) PATCH=$((PATCH + 1)) ;;
esac

VERSION="${MAJOR}.${MINOR}.${PATCH}"
echo "Releasing v${VERSION} (bumped from ${CURRENT_VERSION} via the ${VERSION_LABEL} label)"

git config user.name "NeetoBot"
git config user.email "bot@neeto.com"
git config core.hooksPath /dev/null

git fetch origin main
git checkout main
git pull --ff-only origin main

echo "$VERSION" > VERSION
git add VERSION
git commit -m "Bump version to $VERSION"
git push origin HEAD:main

if git rev-parse "v${VERSION}" >/dev/null 2>&1; then
  echo "Tag v${VERSION} already exists. Skipping tag creation."
else
  git tag -a "v${VERSION}" -m "Release v${VERSION}"
  git push origin "v${VERSION}"
  echo "Pushed tag v${VERSION}"
fi

echo "Opening the version-bump pull requests on every product CLI..."
bash .scripts/bump.sh "v${VERSION}"

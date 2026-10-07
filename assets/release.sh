#!/bin/bash
set -euo pipefail

CONFIG_FILE="${NEETO_CLI_CONFIG:-.neeto-cli.yml}"
SYNC_PKG="github.com/neetozone/neeto-cli-commons/cmd/neeto-cli-sync"

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

for tool in git go gh aws goreleaser sha256sum; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "Required tool not found on PATH: $tool" >&2
    exit 1
  fi
done

if [ ! -f "$CONFIG_FILE" ]; then
  echo "No ${CONFIG_FILE} in $(pwd). The release script reads every product value from it." >&2
  exit 1
fi

PR_NUMBER=$(gh pr list --state merged --base main --limit 50 --json number,mergedAt --jq "sort_by(.mergedAt) | last | .number")
echo "Last merged PR number: $PR_NUMBER"

if [ -z "$PR_NUMBER" ]; then
  echo "No merged PR found. Skipping release."
  exit 0
fi

PR_LABELS=$(gh pr view "$PR_NUMBER" --json labels --jq ".labels[].name" | tr "\n" " ")
echo "PR labels: $PR_LABELS"

VERSION_LABEL=""
for label in major minor patch; do
  if echo "$PR_LABELS" | grep -qw "$label"; then
    VERSION_LABEL="$label"
    break
  fi
done

echo "Version label selected: $VERSION_LABEL"

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
  major)
    MAJOR=$((MAJOR + 1))
    MINOR=0
    PATCH=0
    ;;
  minor)
    MINOR=$((MINOR + 1))
    PATCH=0
    ;;
  patch)
    PATCH=$((PATCH + 1))
    ;;
esac

VERSION="${MAJOR}.${MINOR}.${PATCH}"
echo "Releasing version: $VERSION (bumped from $CURRENT_VERSION via $VERSION_LABEL label)"

# neeto-cli-commons is a private module, so the toolchain cannot fetch it
# without both of these.
export GOPRIVATE="github.com/neetozone/*"
if [ -n "${GITHUB_TOKEN:-}" ]; then
  git config --global url."https://x-access-token:${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"
fi

PRODUCT_ENV=$(go run "$SYNC_PKG" --config "$CONFIG_FILE" --env)
eval "$PRODUCT_ENV"
echo "Releasing ${PRETTY_NAME} (${BINARY_NAME}) from ${REPO_NAME}"

RENDER_DIR=$(mktemp -d)
trap 'rm -rf "$RENDER_DIR"' EXIT INT TERM

echo "Rendering release assets from neeto-cli-commons..."
go run "$SYNC_PKG" --config "$CONFIG_FILE" --out "$RENDER_DIR"

git config user.name "NeetoBot"
git config user.email "bot@neeto.com"
git config core.hooksPath /dev/null

if git ls-remote --exit-code --tags origin "refs/tags/v${VERSION}" >/dev/null 2>&1; then
  echo "Tag v${VERSION} already exists. Skipping tag creation."
else
  git tag -a "v${VERSION}" -m "Release v${VERSION}"
  echo "Created tag v${VERSION}"
  git push origin "v${VERSION}"
  echo "Pushed tag v${VERSION}"
fi

export GORELEASER_CURRENT_TAG="v${VERSION}"
export TAP_GITHUB_TOKEN="${TAP_GITHUB_TOKEN:-${GITHUB_TOKEN:-}}"

if [ -z "${AWS_REGION:-}" ]; then
  BUCKET_REGION=$(aws s3api get-bucket-location --bucket "$S3_BUCKET" --query 'LocationConstraint' --output text 2>/dev/null || true)
  case "$BUCKET_REGION" in
    ""|None|null) BUCKET_REGION="us-east-1" ;;
  esac
  export AWS_REGION="$BUCKET_REGION"
fi
export AWS_DEFAULT_REGION="${AWS_DEFAULT_REGION:-$AWS_REGION}"

echo "GoReleaser version:"
goreleaser --version || true

echo "Running goreleaser release..."
echo "The blobs pipe uploads the versioned artifacts before the Homebrew formula"
echo "is written, so the formula never points at objects that are not there yet."
goreleaser release --clean --config "${RENDER_DIR}/.goreleaser.yml"
echo "GoReleaser release complete."

echo "Preparing installer archives and SHA256SUMS..."
rm -rf dist/installer-archives
mkdir -p dist/installer-archives
cp "dist/${REPO_NAME}_${VERSION}_linux_amd64.tar.gz" "dist/installer-archives/${BINARY_NAME}_linux_amd64.tar.gz"
cp "dist/${REPO_NAME}_${VERSION}_linux_arm64.tar.gz" "dist/installer-archives/${BINARY_NAME}_linux_arm64.tar.gz"
cp "dist/${REPO_NAME}_${VERSION}_darwin_amd64.tar.gz" "dist/installer-archives/${BINARY_NAME}_macos_amd64.tar.gz"
cp "dist/${REPO_NAME}_${VERSION}_darwin_arm64.tar.gz" "dist/installer-archives/${BINARY_NAME}_macos_arm64.tar.gz"
cp "dist/${REPO_NAME}_${VERSION}_windows_amd64.zip" "dist/installer-archives/${BINARY_NAME}_windows_amd64.zip"
cp "dist/${REPO_NAME}_${VERSION}_windows_arm64.zip" "dist/installer-archives/${BINARY_NAME}_windows_arm64.zip"
(cd dist/installer-archives && sha256sum "${BINARY_NAME}"_* > SHA256SUMS)
cat dist/installer-archives/SHA256SUMS

aws s3 cp dist/installer-archives/ "${S3_BASE}/v${VERSION}/" --recursive

echo "Generating versioned installer scripts..."
VERSIONED_URL="${S3_HTTPS_BASE}/v${VERSION}"
LATEST_URL="${S3_HTTPS_BASE}/latest"
rm -rf dist/installers
mkdir -p dist/installers
for installer in install.sh install.ps1 install.cmd; do
  sed "s|${LATEST_URL}|${VERSIONED_URL}|g" "${RENDER_DIR}/installers/${installer}" > "dist/installers/${installer}"
  aws s3 cp "dist/installers/${installer}" "${S3_BASE}/v${VERSION}/${installer}" --content-type "text/plain"
done

echo "Staging the latest/ directory..."
rm -rf dist/latest
mkdir -p dist/latest
cp dist/installer-archives/* dist/latest/
cp "${RENDER_DIR}/installers/install.sh" dist/latest/install.sh
cp "${RENDER_DIR}/installers/install.ps1" dist/latest/install.ps1
cp "${RENDER_DIR}/installers/install.cmd" dist/latest/install.cmd

aws s3 cp dist/latest/ "${S3_BASE}/latest/" --recursive --exclude "install.*"
for installer in install.sh install.ps1 install.cmd; do
  aws s3 cp "dist/latest/${installer}" "${S3_BASE}/latest/${installer}" --content-type "text/plain"
done

echo "Pruning stale objects from latest/..."
KEEP=$(cd dist/latest && ls -1)
EXISTING=$(aws s3 ls "${S3_BASE}/latest/" | awk '{print $4}' || true)
printf '%s\n' "$EXISTING" | while read -r key; do
  if [ -z "$key" ]; then
    continue
  fi
  if ! printf '%s\n' "$KEEP" | grep -qxF "$key"; then
    echo "  removing stale ${key}"
    aws s3 rm "${S3_BASE}/latest/${key}"
  fi
done
echo "S3 upload complete."

BUMP_BRANCH="bump-version-to-${VERSION}"
if [ -n "$(gh pr list --head "$BUMP_BRANCH" --state open --json number --jq '.[].number')" ]; then
  echo "A pull request bumping VERSION to ${VERSION} is already open."
else
  git fetch origin main
  git checkout -B "$BUMP_BRANCH" origin/main
  echo "$VERSION" > VERSION
  git add VERSION
  git commit -m "Bump version to $VERSION"
  git push --force origin "$BUMP_BRANCH"
  gh pr create --base main --head "$BUMP_BRANCH" --label instant-mergepr \
    --title "Bump version to $VERSION" \
    --body "Records v${VERSION} in VERSION after the release. main only takes changes through pull requests, so the release pipeline raises the bump here and instant-mergepr merges it once CI passes."
fi

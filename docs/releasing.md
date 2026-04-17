# Releasing

The template repo releases the `neeto-cli-gen` binary itself. Generated
repos have their own independent release pipelines (see the generated
`.scripts/release.sh`).

## Version bump

1. Edit `VERSION` at the repo root.
2. Edit `internal/version/embedded_version.txt` to match (this is what the
   built binary reports when no ldflag override is present).
3. Commit and open a PR labeled `major` / `minor` / `patch`.

## Cutting a release (local)

```bash
git tag v$(cat VERSION)
goreleaser release --clean
```

## Install paths

- Primary: `go install github.com/neetozone/neeto-cli-template/cmd/neeto-cli-gen@latest`
- GitHub Releases: attached archives from GoReleaser.

## Consumers of a template version

Every generated repo writes the generator's semver into `.template-version`.
To check which template a repo came from:

```bash
cat /path/to/generated-repo/.template-version
```

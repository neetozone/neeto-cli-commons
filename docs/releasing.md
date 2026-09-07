# Releasing neeto-cli-commons

Merging a pull request into `main` with a `major`, `minor` or `patch` label
releases this repo and rolls the new version out to every product CLI. A merge
with no version label does nothing.

`.neetoci/release.yml` runs `.scripts/release.sh` on every push to `main`. That
script:

1. Reads the version label off the last merged pull request. No label, no release.
2. Bumps `VERSION` accordingly, commits it as `Bump version to X.Y.Z` and pushes
   to `main`. That commit is skipped on the next run, so the pipeline cannot loop.
3. Tags the commit `vX.Y.Z` and pushes the tag.
4. Runs `.scripts/bump.sh vX.Y.Z`.

## The fan-out

`.scripts/bump.sh` walks `products.txt` and, for each product repo, on a branch
named `bump-neeto-cli-commons-vX.Y.Z`:

- points `go.mod` at the new version and runs `go mod tidy`
- sets `commons_version` in `.neeto-cli.yml`, which is what makes the product's
  release pipeline fetch the shared release script from the new tag rather than
  from `main`
- re-runs `neeto-cli-sync`, so every generated file picks up the change
- runs `gofmt`, `go build`, `go vet`, `go test` and `neeto-cli-sync --check`,
  and refuses to open a pull request if any of them fail
- opens a pull request labelled `patch`

A repo already on that version is reported as "already current" and left alone.
A repo with no `.neeto-cli.yml`, or whose `go.mod` does not require this module,
is reported as "not ported yet" and fails the run rather than passing quietly.

## Product versions

Each product's own version is bumped by its own release pipeline, not here. The
`patch` label on the pull request bump.sh opens is what does it: when that PR is
merged, the product's `.neetoci/release.yml` bumps its `VERSION`, tags it, builds
the binaries and uploads them. So one labelled merge here ends with eleven
released CLIs, each with its own patch bump.

Reviewing and merging those eleven pull requests is the only manual step.

## Running the fan-out by hand

```bash
bash .scripts/bump.sh v1.4.0
```

Needs `GITHUB_TOKEN` in the environment, and `git`, `go` and `gh` on PATH.

## The generator binary

`.goreleaser.yml` builds `neeto-cli-gen` from the nested `gen` module. Nothing
runs it yet — the generator is used through `go run` or `go install`. Wire it
into `.scripts/release.sh` if the binary should be published.

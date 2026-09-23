# Releasing neeto-cli-commons

Merging a pull request into `main` with a `major`, `minor` or `patch` label
releases this repo and rolls the new version out to every product CLI. A merge
with no version label does nothing.

`.neetoci/release.yml` runs `.scripts/release.sh` on every push to `main`. That
script:

1. Reads the version label off the last merged pull request. No label, no release.
2. Works out the next version from `VERSION` or the latest `v*` tag, whichever is
   higher, so a bump pull request that has not merged yet cannot cause a repeat.
3. Tags the released commit `vX.Y.Z` and pushes the tag.
4. Opens a `Bump version to X.Y.Z` pull request that updates `VERSION`, labelled
   `mergepr` and nothing else. `main` only takes changes through pull requests.
   When it merges, the pipeline sees the bump commit and skips, so it cannot loop.
5. Publishes the `neeto-cli-gen` binaries for that tag.
6. Runs `.scripts/bump.sh vX.Y.Z`.

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
the binaries and uploads them. So one labelled merge here ends with thirteen
released CLIs, each with its own patch bump.

Reviewing and merging those thirteen pull requests is the only manual step.

## First time through

The fan-out needs every product already ported to this module. Until that has
happened, merge into `main` here **without** a version label: an unlabelled
merge releases nothing. Once this module and all thirteen product pull requests
are on `main`, the next labelled merge cuts the first real release and rolls it
out. A repo that is not ported yet fails the run rather than being skipped
quietly, so a premature labelled merge produces thirteen noisy failures after the
tag has already been pushed.

## Running the fan-out by hand

```bash
bash .scripts/bump.sh v1.4.0
```

Needs `GITHUB_TOKEN` in the environment, and `git`, `go` and `gh` on PATH.

## The generator binary

The release also publishes `neeto-cli-gen`, built from the nested `gen` module,
as a GitHub release on the `vX.Y.Z` tag with linux and darwin builds for amd64
and arm64 plus a checksums file. `go install` keeps working as before; the
binaries are for anyone who would rather not build it.

If goreleaser fails, the run says so and carries on to the product roll-out,
then exits non-zero. The tag is already pushed and the version bump pull request
is open at that point, so re-run `goreleaser release --clean` against the tag rather than
re-running the whole pipeline.

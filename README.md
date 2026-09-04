# Smidja bundle template

The canonical GitHub template for compiled [smidja](https://github.com/digitalygo/smidja) bundle distributions. A bundle is the smidja harness with skills, prompts, configuration defaults, and extensions baked in at build time, shipped as one static binary with zero runtime dependencies.

This template ships one working example of everything a bundle can contain, plus the automation to verify, customize, and release it. It is exclusively for compiled distributions: this repository is not itself a content-only package, it carries no `smidja.json`, and it adds no plugins, subprocess extensions, WASM, or any other dynamic executable extension mechanism. The compiled harness retains its documented content-package feature, so the running binary can still load content-only packages and user or workspace content; the skills, prompts, defaults, and extensions this bundle contributes are compiled into the binary.

For the full guide to authoring bundles against the smidja harness, read the [harness creator guide](https://github.com/digitalygo/smidja/blob/alpha/docs/creating-bundles.md).

## What is inside

- `main.go`, the thin entry point composed through the public `smidja.Run` seam.
- `bundle.go`, the `sdk.Bundle` composition: embedded content, extensions, and configuration defaults.
- `identity.go`, the bundle identity validation used by tests and CI.
- `extensions/example`, one deterministic example extension registering the `bundle_digest` tool, which computes SHA-256 hex digests offline and without credentials.
- `content/skills/example.md`, `content/prompts/example.md`, and `content/config/defaults.env`, embedded at build time and conforming to the current smidja content loader.
- `scripts/build-release.sh`, the release cross-build and checksum script.
- `scripts/customize.sh`, the one command that renames the bundle to your repository.
- `.github/workflows/ci.yml` and `.github/workflows/release.yml`.

## Use this template

Create your repository with the GitHub `Use this template` button, then clone it. Do not edit identity strings by hand: run the customization script once.

```bash
git clone https://github.com/your-name/your-bundle.git
cd your-bundle
scripts/customize.sh github.com/your-name/your-bundle
```

The script rewrites the go module path, the bundle origin, and the go imports, then builds to verify. It refuses to run on a dirty working tree. After it finishes:

- Adjust the prose in `README.md` and `AGENTS.md` to your bundle name.
- Commit the rename as your first commit.
- Push a `v*` tag when you want your first release.

CI enforces the identity contract with `GITHUB_REPOSITORY`: any repository generated from this template fails the identity test with a remediation message until the origin matches the repository it lives in. The pristine template passes because its origin already matches `github.com/digitalygo/smidja-bundle-template`.

## Local verification

```bash
gofmt -l .
go vet ./...
go test -count=1 -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
bash -n scripts/*.sh
```

The first `go` command that resolves the harness pin may download `github.com/digitalygo/smidja` into the Go module cache, so first-time verification can touch the network; later runs resolve the pinned commit from the local module cache.

The test suite covers bundle identity, embedded content, defaults parsing, the example extension surface, build identity, the customization script, and the release script including its four cross builds and checksum verification.

A plain local build stays coherent without any flags:

```bash
go build -o smidja-bundle-template .
./smidja-bundle-template version --json
```

This reports the module path as origin, `dev` as version, and `none` as commit.

## Release

Push a `v*` tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow builds `CGO_ENABLED=0` static binaries for linux and darwin on amd64 and arm64, injects the origin, the tag, and the exact source commit through ldflags, generates a sorted `checksums.txt`, verifies it, smokes the linux amd64 binary, and publishes the GitHub release.

Assets are named exactly:

- `smidja-linux-amd64`
- `smidja-linux-arm64`
- `smidja-darwin-amd64`
- `smidja-darwin-arm64`
- `checksums.txt`

These names are a contract, not a convention: the harness self-updater looks up `smidja-<goos>-<goarch>` and `checksums.txt` in the release of the repository baked into the binary.

You can also run the release build locally:

```bash
scripts/build-release.sh v0.1.0 dist
cd dist && sha256sum -c checksums.txt
```

## Install

Download `checksums.txt` and the asset for your platform from the latest release of your bundle repository, keeping the exact published filenames, verify the asset against its manifest line, and then put it on your path:

```bash
curl -fsSL -O https://github.com/your-name/your-bundle/releases/latest/download/checksums.txt
curl -fsSL -O https://github.com/your-name/your-bundle/releases/latest/download/smidja-linux-amd64
grep ' smidja-linux-amd64$' checksums.txt | sha256sum -c -
install -m 0755 smidja-linux-amd64 ~/.local/bin/smidja
```

Replace `linux-amd64` with the asset matching your platform. The asset must keep its published filename because the checksum manifest references it by name; renaming first would leave the download unverifiable.

## Self-update

The binary updates itself from its own origin repository:

```bash
smidja update --check
smidja update
```

Because the release pipeline injects your bundle origin and asset names into the binary, `smidja update` targets your repository's releases, downloads the `smidja-<goos>-<goarch>` asset, verifies it against `checksums.txt`, and replaces the running executable atomically on Linux.

One limitation: the harness update client does not run the optional self-check that would execute the downloaded binary and confirm it reports the expected origin and version before the swap. The capability exists in the harness update package, but the `smidja update` command constructs its client without it, so checksum verification against the published `checksums.txt` is the integrity guarantee for updates.

## Trust boundary

- The binary is the trust root. Skills, prompts, defaults, and extensions are compiled into it, so everything they contain is code you chose at build time. Review the template before customizing and pin the harness by commit.
- Embedded skills and prompts are content that enters the model context. They shape agent behavior but cannot execute anything by themselves.
- Workspace content is untrusted by default: the harness only reads workspace tier content when workspace trust is explicitly enabled.
- Updates verify the downloaded binary against the `checksums.txt` published with the release and replace the binary atomically. The checksum manifest is the update integrity check, not a trust anchor: a checksum published beside a binary shows the download is consistent with the release, not who published it. Publisher authenticity comes from repository and release provenance, meaning you fetch releases only from the repository the binary names. The harness does not run the optional origin and version self-check. The release workflow holds the only write access and runs exclusively on `v*` tags with no secrets.

## Version pinning

The harness dependency is pinned in `go.mod` to an immutable go module version resolving to an exact commit:

```text
github.com/digitalygo/smidja v0.2.1-0.20260904203549-a8a32dc552af
```

That pseudo-version is commit `a8a32dc552af5a3088f0c3e54a47d99b7a86bf11` of the smidja harness. `identity_test.go` rejects branch names such as `alpha` and requires either a tagged version or a pseudo-version, so the build stays reproducible.

To move to a newer harness commit, resolve the commit with Go tooling and pin it:

```bash
go get github.com/digitalygo/smidja@a8a32dc552af5a3088f0c3e54a47d99b7a86bf11
go mod tidy
```

Then update `minimumHarness` in `bundle.go` to the base version of the new pin and run the tests.

## Minimum harness

`bundle.go` declares `MinimumHarness: "v0.2.1"`, the lowest harness version this bundle is expected to work with. It is currently declarative rather than runtime-enforced: the harness carries the field but does not refuse to run when the compiled-in harness is older. Keep it in sync with the `go.mod` pin; a test reads both and fails when they drift.

## License

MIT.

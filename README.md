# Smidja runtime package template

Author installable Lid, Verk, skills and tool helpers for the manager/client runtime. This repository no longer builds an alternative Smidja application. The previous sdk.Bundle implementation remains under [legacy](legacy/README.md), with its original tests and pinned dependency.

## Build locally

Requires Go 1.26 and Python 3.10+ for authoring only. Installed packages contain a native helper and do not require Python.

    go test -coverprofile=coverage.out ./...
    python3 -m unittest discover -s scripts
    bash scripts/build-packages.sh dist https://publisher.example/packages/example.coding/1.0.0

This builds four independent archives:

    dist/linux-amd64/package.zip
    dist/linux-arm64/package.zip
    dist/darwin-amd64/package.zip
    dist/darwin-arm64/package.zip

Each sibling release.json contains id, version, archive_url and the archive SHA256. The source smidja.json is an authoring manifest; the builder generates the final file inventory and fixes each archive to exactly one target. ZIP entries are deterministic regular files, with executable helpers explicitly marked. No install hook runs.

The example URL is deliberately not live. Building does not publish or install anything. CI only verifies and builds artifacts; release publication requires a separate explicit action.

## Customize

Edit package/smidja.json identity, publisher, stable x.y.z version and minimum Smidja version. Update every namespaced Lid, tool, Verk ID and reference consistently. Add JSON documents under package and list them in the manifest. Helpers are built into a temporary staging directory, never committed as binaries.

The sample PR review accepts supplied evidence, not a PR URL to fetch. Its input object requires request and repository (nonempty strings), base_sha and head_sha (exactly 40 characters), and diff (a string, possibly empty). Optional focus is a string and checks accepts any JSON value. Extra properties are rejected. SHA length is validated, not hexadecimal syntax. The owner can always launch it; allowed_invokers explicitly includes ceo for agent launches.

Every Lid agent requires a nonempty name, explicit model and instructions. Agent names must be unique within a Lid and at most 255 UTF-8 bytes. The sample selects openai/gpt-6.1-terra, matching the organization package; replace it with a model available to your Envai account before publishing. There is no implicit model fallback during Lid materialization.

The sample PR review has four phases: context, analysis, verification and synthesis. Its single Lid is intentionally a neutral starter, not an organizational coding standard. The sample echo tool demonstrates JSON stdin/stdout only. To inspect actual repositories, configure the Lid's appropriate runtime tool names and instructions; this package does not itself provide a GitHub integration. The manager owns task creation, full phase outputs, binary blocking classification and resume; none of those are recreated by the helper.

Tool command paths beginning with ./ must name an inventoried file. Arguments are sent as a JSON object on stdin. Successful helpers emit JSON on stdout; errors write stderr and exit nonzero. The runtime wraps stdout as content, rather than assuming that a successful process means its JSON says success. Tool environment maps contain environment-variable references, never secret values. Missing credentials fail at invocation, not package installation; packages do not provision accounts.

Optional timeout_seconds is an integer from 0 through 600; omitted/0 uses the runtime's 30-second default. Use a longer explicit limit only where the service needs it. This field requires the current revamp runtime and is not supported by legacy releases.

## Install and updates

Choose the explicit feed for the installation machine, for example:

    {"url":"https://publisher.example/packages/example.coding/linux-amd64/release.json"}

For a private source add token_env naming a locally configured credential variable. Do not put tokens in URLs or archives. A real update feed should remain at one stable target-specific URL while its release document points to a versioned immutable archive. The build command's base URL is that immutable archive base, not the update-feed URL.

The current runtime does not select platform assets automatically. Never install a Linux archive on macOS or an amd64 archive on arm64. Both manager and client scopes are supported. The manager or client exposes capabilities only after it accepts the manifest, integrity checks and runtime catalog. Lid names are scoped to the installation by Smidja, not hardcoded to a laptop in the package.

Installing trusts the chosen publisher's future updates. Extra agent-created extensions still use the separate approval flow. Dependencies are exact version/digest pins and must already be installed; this template's closed example has none. Changing an installed package's files is not customization: publish a new immutable version, or create private configuration outside the package.

No package releases exist from this draft. requires_smidja >=1.0.0 is an example compatibility floor for the new runtime, not a claim that Smidja 1.0.0 has been released. The runtime enforces this floor: older versions and unversioned development builds (0.0.0) reject these archives. Set the floor to the actual compatible revamp release before publishing; do not lower it solely to bypass the installation check.

## Validation

scripts/package_builder.py is standard-library-only and reusable:

    build(source_directory, output_directory, target, archive_base_url)
    validate_archive(zip_bytes)

The authoring validator checks local catalog references, IDs, environment references and file integrity. It intentionally requires referenced Lid/tools to be present in this package; cross-package references need explicit dependency-aware authoring support rather than guessed IDs. The authoritative runtime remains Smidja's schema-1 installer and capability loader.

Legacy tests can still be run independently with cd legacy && go test ./...; they are not runtime-package tests. Moving them did not remove their historical assertions.

## Verk input and invocation contracts

A Verk may supply input_schema. Omission preserves compatibility with an unconstrained schema, while actual Verk inputs remain JSON objects. This example uses a strict object schema for supplied PR evidence. The supported schema keywords are type, properties, required, additionalProperties (boolean only), items, enum, const, minimum, maximum, minLength, maxLength, minItems, maxItems, description and title. Unsupported keywords, including pattern, $ref and $schema, are rejected. Schema nesting is limited to 32 levels and encoded schemas to 64 KiB; the runtime limits inputs to 1 MiB.

allowed_invokers authorizes agents, not the owner: omitted means owner plus Smidr, an explicit empty list means owner only, and entries may be ceo or `team:<stable team reference>`. Entries must be unique. This package explicitly chooses ["ceo"]. Stable references are checked at invocation; installing a package does not grant arbitrary agents access.

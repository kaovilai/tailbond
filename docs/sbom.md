# SBOM: what's inside each release file

Every [release](https://github.com/kaovilai/tailbond/releases) has a
software bill of materials for all of its files:

- `tailbond-<version>.spdx.json`: SPDX 2.3, the attested one
- `tailbond-<version>.cdx.json`: the same, converted to CycloneDX

There is one package per release file (name and SHA256, as in `SHA256SUMS`),
and under each one what it contains:

| Inside | Where it comes from |
|---|---|
| tailbondd's Go modules and Go version | syft, from the build info in the binary |
| mqvpn, xquic, BoringSSL, lwIP, yyjson (in every file that ships mqvpn) | [`scripts/sbom/mqvpn-components.json`](../scripts/sbom/mqvpn-components.json); versions from the pinned submodule commits and their sources |
| libevent (Linux bundles, ipks, macOS bundle: linked statically) | the version string in the shipped `mqvpn` binary |
| The Android app's Maven libraries | Gradle, `releaseRuntimeClasspath` (what the release APK ships) |
| The APK inside the ipk and rpm | matched by SHA256 to `tailbond-android.apk` (the OpenWrt 25.12 apk carries it too; its container is not extracted, so the ipk is the match) |

syft can't see C libraries linked statically into mqvpn (there's no
package metadata in the binary), so they come from a hand-kept list. The
list holds no versions: CI derives them on every run from the submodules
(`third_party/mqvpn` and its xquic, BoringSSL and lwIP) and from the sources
(mqvpn's `CMakeLists.txt`, lwIP's `init.h`, yyjson's header). It fails when
a version can't be derived, and when mqvpn gains a `third_party` directory
the list doesn't mention. The Fedora RPM's mqvpn links Fedora's own libevent,
which the RPM depends on rather than ships.

## Check a file

```sh
gh release download vX.Y.Z -R kaovilai/tailbond -p tailbond-linux-x86_64.tar.gz -p 'tailbond-*.spdx.json'
gh attestation verify tailbond-linux-x86_64.tar.gz -R kaovilai/tailbond --predicate-type https://spdx.dev/Document
```

The attestation binds the SBOM to every file in `SHA256SUMS`. Add
`--format json --jq '.[0].verificationResult.statement.predicate'` to print
the attested SBOM itself. To look up known vulnerabilities:
`grype sbom:tailbond-X.Y.Z.spdx.json`. CI runs grype on every build too, as
a report in the run summary that never fails the build.

## Dependabot alerts: what's covered

GitHub raises Dependabot alerts only for the default branch's dependency
graph. Releases are tagged from `main` ([updates](updates.md)), so on every
push to `main` CI submits the SBOM of that build (job `dependency-snapshot`,
one manifest per release file, which includes the libraries of the APK
the ipk and rpm ship) to the dependency graph. Alerts then cover
what is actually shipped:

| Dependencies | In the graph | Dependabot alerts |
|---|---|---|
| Go modules | yes (`go.mod`, and as built) | yes |
| Android Maven libraries | yes, from this submission only | yes |
| GitHub Actions | yes (workflow files) | yes |
| Go standard library | yes, as built (`pkg:golang/stdlib`) | only where GitHub has an advisory for it; grype always checks |
| mqvpn, xquic, BoringSSL, lwIP, yyjson, libevent | yes, as `pkg:github/...` | no: GitHub has no advisory ecosystem for C libraries. grype matches libevent and lwIP by CPE; follow BoringSSL and xquic upstream |

A snapshot submitted for any other branch (CI on `workflow_dispatch`)
feeds dependency review for pull requests, not alerts.

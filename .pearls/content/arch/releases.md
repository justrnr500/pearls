# Release Pipeline

## How Releases Work

1. Tag a commit: \`git tag v0.1.0 && git push --tags\`
2. GitHub Actions runs GoReleaser inside \`goreleaser/goreleaser-cross\` Docker image
3. Builds 4 binaries (macOS arm64/amd64, Linux arm64/amd64) with CGO enabled
4. Creates GitHub Release with tarballs, checksums, and changelog

## Targets

| OS | Arch | Archive |
|----|------|---------|
| macOS | arm64 | \`pearls_<ver>_darwin_arm64.tar.gz\` |
| macOS | amd64 | \`pearls_<ver>_darwin_amd64.tar.gz\` |
| Linux | arm64 | \`pearls_<ver>_linux_arm64.tar.gz\` |
| Linux | amd64 | \`pearls_<ver>_linux_amd64.tar.gz\` |

Each archive contains both \`pearls\` and \`pl\` binaries.

## Versioning

\`var version = "dev"\` in \`cmd/pearls/main.go\` and \`cmd/pl/main.go\`.
GoReleaser injects the real tag via \`-ldflags "-X main.version={{.Version}}"\`.
\`pearls --version\` outputs the version.

## Install Script

\`\`\`bash
curl -fsSL https://raw.githubusercontent.com/justrnr500/pearls/main/scripts/install.sh | bash
\`\`\`

Detects OS/arch, downloads the right tarball from the latest GitHub Release,
extracts to \`/usr/local/bin\` (or \`~/.local/bin\` without sudo).

## Key Files

- \`.goreleaser.yaml\` — build configuration (targets, archives, ldflags)
- \`.github/workflows/release.yml\` — CI workflow triggered by tag push
- \`scripts/install.sh\` — user-facing installer
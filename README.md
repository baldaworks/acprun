# acprun

[![Test](https://github.com/baldaworks/acprun/actions/workflows/test.yml/badge.svg)](https://github.com/baldaworks/acprun/actions/workflows/test.yml)
[![Lint](https://github.com/baldaworks/acprun/actions/workflows/lint.yml/badge.svg)](https://github.com/baldaworks/acprun/actions/workflows/lint.yml)
[![Security](https://github.com/baldaworks/acprun/actions/workflows/security.yml/badge.svg)](https://github.com/baldaworks/acprun/actions/workflows/security.yml)
[![Latest release](https://img.shields.io/github/v/release/baldaworks/acprun)](https://github.com/baldaworks/acprun/releases/latest)
[![npm version](https://img.shields.io/npm/v/%40baldaworks%2Facprun)](https://www.npmjs.com/package/@baldaworks/acprun)
[![License: MIT](https://img.shields.io/github/license/baldaworks/acprun)](LICENSE)

## ACP Registry client and agent runner

`acprun` is a CLI for discovering, inspecting, resolving, and running agents published in the [Agent Client Protocol (ACP) Registry](https://agentclientprotocol.com). It supports native binaries, `npx`, and `uvx`.

It transparently handles distribution resolution across:
- **Binary Archives**: Detects host OS and CPU architecture (`linux-x86_64`, `linux-aarch64`, `darwin-aarch64`, `darwin-x86_64`, `windows-x86_64`, `windows-aarch64`), downloads `.zip` or `.tar.gz` archives, validates optional SHA256 integrity checksums, safely extracts to local user cache (with strict Zip Slip protection), sets `0755` executable permissions, and builds the command vector.
- **NPX Packages**: Formats `["npx", "-y", <package>, <args...>]` and sets environment variables.
- **UVX Packages**: Formats `["uvx", <package>, <args...>]` and sets environment variables.
- **Offline Resilient Caching**: Caches registry manifests and downloaded binaries under `os.UserCacheDir()/acprun/` for fast, offline reuse.

---

## Installation and quick start

Install `acprun` through either npm launcher or as a Go binary.

### npm package (recommended)

For repeated direct shell use, install the npm launcher globally:

```bash
npm install --global acprun@latest
acprun version
```

The equivalent scoped package is also available as `@baldaworks/acprun`:

```bash
npm install --global @baldaworks/acprun@latest
```

For one-shot execution without global installation, run the complete `npx` command:

```bash
npx --yes acprun@latest list
# Equivalent scoped launcher:
npx --yes @baldaworks/acprun@latest list
```

### Go binary

```bash
go install github.com/baldaworks/acprun/cmd/acprun@latest
```

These packages install the `acprun` CLI. When running an agent, `acprun` selects that agent's binary, `npx`, or `uvx` distribution from the Registry.

---

## One-shot agent execution

`acprun` supports **one-shot execution** directly without needing explicit subcommands (similar to `npx` or `uvx`):

```bash
# Run a binary agent
acprun goose

# Run an NPX agent with arguments
acprun cline --acp

# Run a UVX agent
acprun fast-agent -x

# Pass arbitrary flags directly to the agent
acprun amp-acp --verbose
```

---

## Agent discovery and inspection

### `acprun list`
List all agents available in the ACP registry.
```bash
# Formatted table
acprun list

# Filter by distribution format
acprun list --distribution binary

# JSON output
acprun list --format json
```

### `acprun info <agent-id>` (alias: `show`)
Display detailed agent metadata, license, authors, repository, and platform distribution specs.
```bash
acprun info devin
acprun info gemini --json
```

### `acprun resolve <agent-id>`
Dry-run resolution: downloads and extracts binary archives to local cache if needed, and prints the resolved command vector without running the process.
```bash
acprun resolve antigravity-acp --json
acprun resolve goose --platform linux-x86_64
```

---

## Explicit runner subcommands

Use explicit runner subcommands (`run`, `serve`, `exec`, `start`) when you want to avoid any potential ambiguity with management subcommands (e.g. running an agent whose ID is `list` or `cache`):

```bash
# Explicit run
acprun run cursor -- acp

# Using serve alias
acprun serve devin

# Pass extra environment variables
acprun run fast-agent -e FAST_AGENT_MODEL=custom -- -x
```

---

## Cache management

```bash
# Print cache directory path
acprun cache path

# Purge cache
acprun cache clean --all
acprun cache clean --manifests-only
```

---

## CLI reference

### Global Flags
```text
  -r, --registry string    ACP Registry URL (default: official CDN, env: ACP_REGISTRY_URL)
      --cache-dir string   Custom cache directory (default: $USER_CACHE_DIR/acprun, env: ACP_CACHE_DIR)
      --download-timeout duration
                           Binary archive download timeout (default 5m0s)
      --offline            Offline mode: use cached manifests and binaries only
  -v, --verbose            Enable verbose output
  -h, --help               Help for acprun
```

### Subcommands Table
| Command | Description | Aliases |
| --- | --- | --- |
| `acprun <agent-id> [args...]` | One-shot agent resolution and execution | — |
| `acprun run <agent-id>` | Explicit agent process runner | `serve`, `exec`, `start` |
| `acprun list` | List available registry agents | `ls` |
| `acprun info <agent-id>` | Display detailed agent metadata and distributions | `show` |
| `acprun resolve <agent-id>` | Dry-run resolution to command vector | — |
| `acprun cache [path\|clean]` | Inspect or clean local ACP cache | — |
| `acprun version` | Print version and build metadata | — |

---

## Architecture and security

- **Targeted Package Layout**:
  - [`cmd/acprun/`](cmd/acprun/): Binary entrypoint.
  - [`internal/cli/`](internal/cli/): Cobra CLI commands with one-shot dispatch.
  - [`internal/registry/`](internal/registry/): ACP v1 models, HTTP client, and manifest caching.
  - [`internal/resolver/`](internal/resolver/): Host platform detection, safe archive extraction (`.zip`, `.tar.gz`, `.tar.bz2`), and Binary/NPX/UVX distribution resolvers.
  - [`internal/runner/`](internal/runner/): Process execution, stdio binding, and signal forwarding (`SIGINT`, `SIGTERM`).
- **Security & Integrity**:
  - **Zip Slip Prevention**: Validates all archive entry paths stay within the target destination directory before extraction.
  - **SHA256 Verification**: Verifies SHA256 integrity checksums against registry manifests.
  - **Static Binaries**: Built with `CGO_ENABLED=0` for pure Go cross-platform portability.

---

## Distribution

The npm distribution uses CGO-disabled native executables for macOS and Linux on AMD64/ARM64 and Windows AMD64/ARM64 behind the equivalent [`acprun`](https://www.npmjs.com/package/acprun) and [`@baldaworks/acprun`](https://www.npmjs.com/package/@baldaworks/acprun) launchers. Both launchers share the same scoped platform packages.

Description, keywords, and MIT license metadata are defined once in `.omnidist/omnidist.yaml` and inherited by both npm launchers.

### Release setup

Tag-triggered releases stage and verify the npm packages before publishing and attach native binaries to the GitHub release. npm publishing uses GitHub Actions OIDC through omnidist's `publish-auth: trusted` mode. The publish job has `id-token: write`; no npm publishing secret is required.

Sign in locally through npm web authentication with an account that has 2FA enabled and write access to both launcher names and the shared scoped platform packages. Inspect the trusted-publisher plan, then apply it through omnidist:

```bash
npx -y npm@11.16.0 login --auth-type=web --registry=https://registry.npmjs.org
npx -y @omnidist/omnidist@latest npm trust
npx -y @omnidist/omnidist@latest npm trust --apply
```

The plan covers all eight packages and grants publishing access to `baldaworks/acprun` using the workflow filename `omnidist-release.yml`. Complete any browser/2FA prompts from npm. Local web authentication is used to configure trust; GitHub Actions obtains its own short-lived OIDC credentials for releases. See [npm trusted publishing](https://docs.npmjs.com/trusted-publishers/) and the [npm trust prerequisites](https://docs.npmjs.com/cli/v11/commands/npm-trust/).

npm requires a package to exist before its trusted publisher can be configured. A new package name therefore needs its initial publication through omnidist using local web authentication before applying the trust plan. Keep credentials outside the repository.

A failed upload can leave some npm packages published. Retain the release logs and inspect registry state before retrying; omnidist 0.2.0 publishes platform packages first and does not skip existing versions, so replaying a partially published release can fail.

---

## License

`acprun` is released under the [MIT License](LICENSE).

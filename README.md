# acprun

`acprun` is a CLI tool, runner, and Go library for discovering, resolving, and running agents published in the [Agent Client Protocol (ACP) Registry](https://agentclientprotocol.com).

It transparently handles distribution resolution across:
- **Binary Archives**: Detects host OS and CPU architecture (`linux-x86_64`, `linux-aarch64`, `darwin-aarch64`, `darwin-x86_64`, `windows-x86_64`, `windows-aarch64`), downloads `.zip` or `.tar.gz` archives, validates optional SHA256 integrity checksums, safely extracts to local user cache (with strict Zip Slip protection), sets `0755` executable permissions, and builds the command vector.
- **NPX Packages**: Formats `["npx", "-y", <package>, <args...>]` and sets environment variables.
- **UVX Packages**: Formats `["uvx", <package>, <args...>]` and sets environment variables.
- **Offline Resilient Caching**: Caches registry manifests and downloaded binaries under `os.UserCacheDir()/acprun/` for fast, offline reuse.

---

## Installation

### Via Go
```bash
go install github.com/baldaworks/acprun/cmd/acprun@latest
```

### Via NPX (Omnidist)
```bash
npx -y @baldaworks/acprun list
```

---

## Quick Start & One-Shot Mode

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

## CLI Reference

### Global Flags
```text
  -r, --registry string    ACP Registry URL (default: official CDN, env: ACP_REGISTRY_URL)
      --cache-dir string   Custom cache directory (default: $USER_CACHE_DIR/acprun, env: ACP_CACHE_DIR)
      --offline            Offline mode: use cached manifests and binaries only
  -v, --verbose            Enable verbose output
  -h, --help               Help for acprun
```

### Commands

#### `acprun list`
List all agents available in the ACP registry.
```bash
# Formatted table
acprun list

# Filter by distribution format
acprun list --distribution binary

# JSON output
acprun list --format json
```

#### `acprun info <agent-id>` (alias: `show`)
Display detailed agent metadata, license, authors, repository, and platform distribution specs.
```bash
acprun info devin
acprun info gemini --json
```

#### `acprun resolve <agent-id>`
Dry-run resolution: downloads and extracts binary archives to local cache if needed, and prints the resolved command vector without running the process.
```bash
acprun resolve antigravity-acp --json
acprun resolve goose --platform linux-x86_64
```

#### `acprun run <agent-id> [-- extra-args...]` (aliases: `serve`, `exec`, `start`)
Explicit execution form to run any agent without potential name collision (e.g. running an agent named `list` or `cache`).
```bash
# Explicit run
acprun run cursor -- acp

# Using serve alias
acprun serve devin

# Pass extra environment variables
acprun run fast-agent -e FAST_AGENT_MODEL=custom -- -x
```

#### `acprun cache`
Manage local ACP cache directory.
```bash
# Print cache directory path
acprun cache path

# Purge cache
acprun cache clean --all
acprun cache clean --manifests-only
```

#### `acprun version`
Print version and build information.
```bash
acprun version
```

---

## Architecture & Security

- **Targeted Internal Package Layout**:
  - `cmd/acprun/`: Binary entrypoint.
  - `internal/cli/`: Cobra CLI commands with one-shot dispatch.
  - `internal/registry/`: ACP v1 models, HTTP client, and manifest caching.
  - `internal/resolver/`: Host platform detection, safe archive extraction (`.zip`, `.tar.gz`, `.tar.bz2`), and Binary/NPX/UVX distribution resolvers.
  - `internal/runner/`: Process execution, stdio binding, and signal forwarding (`SIGINT`, `SIGTERM`).
- **Security & Integrity**:
  - **Zip Slip Prevention**: Validates all archive entry paths stay within the target destination directory before extraction.
  - **SHA256 Verification**: Verifies SHA256 integrity checksums against registry manifests.
  - **Static Binaries**: Built with `CGO_ENABLED=0` for pure Go cross-platform portability.

---

## License

Apache-2.0 / MIT

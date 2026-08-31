// Package resolver resolves ACP agent distributions into executable command vectors.
package resolver

import (
	"fmt"
	"runtime"
	"strings"
)

// Supported ACP platform identifiers.
const (
	PlatformLinuxX86_64   = "linux-x86_64"
	PlatformLinuxAarch64  = "linux-aarch64"
	PlatformDarwinX86_64  = "darwin-x86_64"
	PlatformDarwinAarch64 = "darwin-aarch64"
	PlatformWindowsX86_64 = "windows-x86_64"
	PlatformWindowsAarch64 = "windows-aarch64"
)

// DetectCurrentPlatform detects host OS and architecture and returns the canonical ACP platform key.
func DetectCurrentPlatform() (string, error) {
	return MapPlatform(runtime.GOOS, runtime.GOARCH)
}

// MapPlatform converts a GOOS and GOARCH pair into a canonical ACP platform key.
func MapPlatform(goos, goarch string) (string, error) {
	osKey := strings.ToLower(strings.TrimSpace(goos))
	archKey := strings.ToLower(strings.TrimSpace(goarch))

	var canonOS string
	switch osKey {
	case "linux":
		canonOS = "linux"
	case "darwin", "macos", "osx":
		canonOS = "darwin"
	case "windows", "win":
		canonOS = "windows"
	default:
		return "", fmt.Errorf("unsupported operating system: %s", goos)
	}

	var canonArch string
	switch archKey {
	case "amd64", "x86_64", "x64":
		canonArch = "x86_64"
	case "arm64", "aarch64":
		canonArch = "aarch64"
	default:
		return "", fmt.Errorf("unsupported architecture: %s", goarch)
	}

	return fmt.Sprintf("%s-%s", canonOS, canonArch), nil
}

// NormalizePlatform takes an arbitrary platform string (which might use Go arch or ACP arch) and returns canonical form.
func NormalizePlatform(platform string) (string, error) {
	platform = strings.TrimSpace(strings.ToLower(platform))
	parts := strings.Split(platform, "-")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid platform string format %q, expected <os>-<arch>", platform)
	}
	return MapPlatform(parts[0], parts[1])
}

package resolver

import (
	"testing"
)

func TestMapPlatform(t *testing.T) {
	tests := []struct {
		goos     string
		goarch   string
		expected string
		wantErr  bool
	}{
		{"linux", "amd64", "linux-x86_64", false},
		{"linux", "x86_64", "linux-x86_64", false},
		{"linux", "arm64", "linux-aarch64", false},
		{"linux", "aarch64", "linux-aarch64", false},
		{"darwin", "arm64", "darwin-aarch64", false},
		{"darwin", "amd64", "darwin-x86_64", false},
		{"macos", "arm64", "darwin-aarch64", false},
		{"windows", "amd64", "windows-x86_64", false},
		{"windows", "arm64", "windows-aarch64", false},
		{"freebsd", "amd64", "", true},
		{"linux", "mips", "", true},
	}

	for _, tt := range tests {
		got, err := MapPlatform(tt.goos, tt.goarch)
		if (err != nil) != tt.wantErr {
			t.Errorf("MapPlatform(%q, %q) error = %v, wantErr %v", tt.goos, tt.goarch, err, tt.wantErr)
			continue
		}
		if got != tt.expected {
			t.Errorf("MapPlatform(%q, %q) = %q, want %q", tt.goos, tt.goarch, got, tt.expected)
		}
	}
}

func TestNormalizePlatform(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"linux-amd64", "linux-x86_64", false},
		{"darwin-arm64", "darwin-aarch64", false},
		{"windows-x64", "windows-x86_64", false},
		{"invalid-format", "", true},
	}

	for _, tt := range tests {
		got, err := NormalizePlatform(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("NormalizePlatform(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.expected {
			t.Errorf("NormalizePlatform(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDetectCurrentPlatform(t *testing.T) {
	platform, err := DetectCurrentPlatform()
	if err != nil {
		t.Fatalf("DetectCurrentPlatform() failed: %v", err)
	}
	if platform == "" {
		t.Errorf("DetectCurrentPlatform() returned empty string")
	}
}

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestBuildReleaseScriptProducesAssetsAndChecksums(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain is required to run the release script")
	}
	dist := filepath.Join(t.TempDir(), "dist")
	build := exec.Command("bash", "scripts/build-release.sh", "v0.0.1-test", dist)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build-release.sh: %v\n%s", err, output)
	}

	wantBinaries := []string{
		"smidja-darwin-amd64",
		"smidja-darwin-arm64",
		"smidja-linux-amd64",
		"smidja-linux-arm64",
	}
	wantAssets := append([]string{"checksums.txt"}, wantBinaries...)
	entries, err := os.ReadDir(dist)
	if err != nil {
		t.Fatalf("read dist: %v", err)
	}
	var gotAssets []string
	for _, entry := range entries {
		gotAssets = append(gotAssets, entry.Name())
	}
	sort.Strings(gotAssets)
	if !reflect.DeepEqual(gotAssets, wantAssets) {
		t.Fatalf("dist contents = %v, want exactly %v", gotAssets, wantAssets)
	}

	checksums := readChecksums(t, filepath.Join(dist, "checksums.txt"))
	var names []string
	for _, entry := range checksums {
		names = append(names, entry.name)
		data, err := os.ReadFile(filepath.Join(dist, entry.name))
		if err != nil {
			t.Fatalf("read asset %s: %v", entry.name, err)
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != entry.digest {
			t.Errorf("checksum for %s = %s, want %s", entry.name, entry.digest, got)
		}
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, wantBinaries) {
		t.Errorf("checksums.txt covers %v, want %v", names, wantBinaries)
	}
	for i := 1; i < len(checksums); i++ {
		if checksums[i-1].name >= checksums[i].name {
			t.Errorf("checksums.txt is not sorted by asset name: %s before %s", checksums[i-1].name, checksums[i].name)
		}
	}

	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		stdout := runBinary(t, filepath.Join(dist, "smidja-linux-amd64"), "version", "--json")
		assertReleaseIdentity(t, stdout, "v0.0.1-test")
	}
}

func TestBuildReleaseScriptRejectsInvalidVersion(t *testing.T) {
	for _, version := range []string{"alpha", "main", "1.2.3", "latest", "v1.2.3garbage", "v1.2.3-", "v1.2.3-beta..1", "v1.2.3-beta.1 extra"} {
		build := exec.Command("bash", "scripts/build-release.sh", version, filepath.Join(t.TempDir(), "dist"))
		if output, err := build.CombinedOutput(); err == nil {
			t.Errorf("build-release.sh %q exited 0, want failure", version)
		} else if len(output) == 0 {
			t.Errorf("build-release.sh %q failed without a diagnostic", version)
		}
	}
}

func TestBuildReleaseScriptAcceptsDocumentedVersionForms(t *testing.T) {
	probe := "set -euo pipefail\nsource scripts/build-release.sh\nvalidate_version \"$1\""
	for _, version := range []string{"v1.2.3", "v1.2.3-beta.1", "v0.0.0-ci", "v0.0.1-test"} {
		output, err := exec.Command("bash", "-c", probe, "build-release.sh", version).CombinedOutput()
		if err != nil {
			t.Errorf("build-release.sh should accept version %q: %v\n%s", version, err, output)
		}
	}
}

func TestBuildReleaseScriptRequiresVersionArgument(t *testing.T) {
	build := exec.Command("bash", "scripts/build-release.sh")
	if output, err := build.CombinedOutput(); err == nil {
		t.Fatal("build-release.sh without arguments exited 0, want failure")
	} else if !strings.Contains(string(output), "usage:") {
		t.Errorf("build-release.sh usage failure output %q does not contain usage", output)
	}
}

type checksumEntry struct {
	digest string
	name   string
}

func readChecksums(t *testing.T, path string) []checksumEntry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var out []checksumEntry
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		parts := strings.Fields(line)
		if len(parts) != 2 {
			t.Fatalf("checksums.txt line %q is not a two field sha256sum line", line)
		}
		if len(parts[0]) != 64 {
			t.Fatalf("checksums.txt line %q has a non sha256 digest", line)
		}
		out = append(out, checksumEntry{digest: parts[0], name: filepath.Base(parts[1])})
	}
	return out
}

func assertReleaseIdentity(t *testing.T, stdout, version string) {
	t.Helper()
	var identity struct {
		Origin  string `json:"origin"`
		Version string `json:"version"`
		Commit  string `json:"commit"`
	}
	if err := json.Unmarshal([]byte(stdout), &identity); err != nil {
		t.Fatalf("release binary version --json output %q is not JSON: %v", stdout, err)
	}
	if identity.Origin != bundleOrigin {
		t.Errorf("release binary origin = %q, want %q so the self-updater targets the creator repository", identity.Origin, bundleOrigin)
	}
	if identity.Version != version {
		t.Errorf("release binary version = %q, want the released tag %q", identity.Version, version)
	}
	wantCommit := sourceCommit(t)
	if identity.Commit != wantCommit {
		t.Errorf("release binary commit = %q, want the source commit %q", identity.Commit, wantCommit)
	}
}

func sourceCommit(t *testing.T) string {
	t.Helper()
	output, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "none"
	}
	return strings.TrimSpace(string(output))
}

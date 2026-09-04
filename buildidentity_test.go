package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/digitalygo/smidja/sdk"
)

func TestBuildInfoMirrorsIdentityVariables(t *testing.T) {
	saved := BuildInfo()
	t.Cleanup(func() {
		bundleOrigin = saved.Origin
		bundleVersion = saved.Version
		bundleCommit = saved.Commit
	})
	bundleOrigin = "github.com/acme/cool-bundle"
	bundleVersion = "v1.2.3"
	bundleCommit = "abc1234"
	got := BuildInfo()
	want := sdk.BuildInfo{Origin: "github.com/acme/cool-bundle", Version: "v1.2.3", Commit: "abc1234"}
	if got != want {
		t.Errorf("BuildInfo() = %+v, want %+v", got, want)
	}
}

func TestBuildInfoPlainBuildDefaults(t *testing.T) {
	got := BuildInfo()
	if got.Origin != bundleOrigin {
		t.Errorf("BuildInfo().Origin = %q, want the bundle origin variable %q", got.Origin, bundleOrigin)
	}
	if got.Version != "dev" {
		t.Errorf("BuildInfo().Version = %q, want dev", got.Version)
	}
	if got.Commit != "none" {
		t.Errorf("BuildInfo().Commit = %q, want none", got.Commit)
	}
}

func TestPlainBuildReportsCoherentIdentity(t *testing.T) {
	binary := buildForSmoke(t, "", "smidja-bundle-template-smoke")
	stdout := runBinary(t, binary, "version", "--json")
	var identity struct {
		Origin  string `json:"origin"`
		Version string `json:"version"`
		Commit  string `json:"commit"`
	}
	if err := json.Unmarshal([]byte(stdout), &identity); err != nil {
		t.Fatalf("version --json output %q is not JSON: %v", stdout, err)
	}
	if identity.Origin != bundleOrigin {
		t.Errorf("plain build origin = %q, want the source default %q", identity.Origin, bundleOrigin)
	}
	if identity.Version != "dev" || identity.Commit != "none" {
		t.Errorf("plain build identity = %+v, want version dev and commit none", identity)
	}
	if got := runBinary(t, binary, "version"); got != "smidja dev\n" {
		t.Errorf("version output = %q, want %q", got, "smidja dev\n")
	}
}

func TestBinaryRejectsUnknownSubcommand(t *testing.T) {
	binary := buildForSmoke(t, "", "smidja-bundle-template-smoke-bad")
	cmd := exec.Command(binary, "definitely-not-a-command")
	cmd.Env = smokeEnv(t)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("unknown subcommand exited 0, want failure; output %q", output)
	}
	if !strings.Contains(string(output), "smidja") {
		t.Errorf("unknown subcommand output %q does not mention smidja", output)
	}
}

func buildForSmoke(t *testing.T, ldflags string, name string) string {
	t.Helper()
	outDir := t.TempDir()
	binary := filepath.Join(outDir, name)
	args := []string{"build", "-o", binary}
	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}
	args = append(args, ".")
	cmd := exec.Command("go", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, output)
	}
	return binary
}

func runBinary(t *testing.T, binary string, args ...string) string {
	t.Helper()
	cmd := exec.Command(binary, args...)
	if binary != "git" {
		cmd.Env = smokeEnv(t)
	}
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s %s: %v", binary, strings.Join(args, " "), err)
	}
	return string(output)
}

func smokeEnv(t *testing.T) []string {
	t.Helper()
	home := t.TempDir()
	env := os.Environ()
	env = append(env, "HOME="+home, "SMIDJA_SESSION_DIR="+filepath.Join(home, "sessions"))
	return env
}

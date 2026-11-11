package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const identityTestPattern = "^(TestValidateIdentity|TestReadGoModIdentity|TestRepositoryIdentityMatchesGoMod|TestMinimumHarnessMatchesPinnedHarnessBase|TestBundleOriginMatchesGitHubRepository)$"

func TestCustomizeScriptRewritesIdentity(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain is required to verify the customization build")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required to exercise the customization dirty-tree guard")
	}
	oldOrigin := currentBundleOrigin(t)
	newOrigin := distinctBundleOrigin(t)
	workDir := t.TempDir()
	copyRepository(t, workDir)
	commitRepository(t, workDir)

	customize := exec.Command("bash", "scripts/customize.sh", newOrigin)
	customize.Dir = workDir
	if output, err := customize.CombinedOutput(); err != nil {
		t.Fatalf("customize.sh: %v\n%s", err, output)
	}

	goMod, err := os.ReadFile(filepath.Join(workDir, "go.mod"))
	if err != nil {
		t.Fatalf("read customized go.mod: %v", err)
	}
	if !strings.Contains(string(goMod), "module "+newOrigin+"\n") {
		t.Errorf("customized go.mod does not declare module %s:\n%s", newOrigin, goMod)
	}

	bundleGo, err := os.ReadFile(filepath.Join(workDir, "bundle.go"))
	if err != nil {
		t.Fatalf("read customized bundle.go: %v", err)
	}
	if !strings.Contains(string(bundleGo), `"`+newOrigin+`"`) {
		t.Errorf("customized bundle.go does not carry origin %s:\n%s", newOrigin, bundleGo)
	}

	assertOriginAbsent(t, workDir, oldOrigin)

	inner := exec.Command("go", "test", "-count=1", "-run", identityTestPattern, "./...")
	inner.Dir = workDir
	inner.Env = repositoryEnv(newOrigin)
	if output, err := inner.CombinedOutput(); err != nil {
		t.Fatalf("focused identity tests inside the customized copy: %v\n%s", err, output)
	}
}

func TestCustomizeScriptRejectsCurrentOriginAndInvalidPaths(t *testing.T) {
	workDir := t.TempDir()
	copyRepository(t, workDir)
	tests := []struct {
		name string
		arg  string
	}{
		{name: "unchanged current origin", arg: currentBundleOrigin(t)},
		{name: "missing repository", arg: "github.com/acme"},
		{name: "not a github origin", arg: "gitlab.com/acme/cool-bundle"},
		{name: "owner longer than 39 characters", arg: "github.com/" + strings.Repeat("a", 40) + "/cool-bundle"},
		{name: "empty argument", arg: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"scripts/customize.sh"}
			if tt.arg != "" {
				args = append(args, tt.arg)
			}
			customize := exec.Command("bash", args...)
			customize.Dir = workDir
			if output, err := customize.CombinedOutput(); err == nil {
				t.Errorf("customize.sh %q exited 0, want failure", tt.arg)
			} else if !strings.Contains(string(output), "usage:") && !strings.Contains(string(output), "must") {
				t.Errorf("customize.sh failure output %q lacks a diagnostic", output)
			}
		})
	}
}

func TestCustomizeScriptRejectsDirtyTree(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain is required to run the customization script")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is required to exercise the customization dirty-tree guard")
	}
	workDir := t.TempDir()
	copyRepository(t, workDir)
	commitRepository(t, workDir)
	if err := os.WriteFile(filepath.Join(workDir, "README.md"), []byte("uncommitted change\n"), 0o644); err != nil {
		t.Fatalf("dirty the copied repository: %v", err)
	}
	customize := exec.Command("bash", "scripts/customize.sh", distinctBundleOrigin(t))
	customize.Dir = workDir
	output, err := customize.CombinedOutput()
	if err == nil {
		t.Fatal("customize.sh on a dirty tree exited 0, want failure")
	}
	if !strings.Contains(string(output), "dirty") {
		t.Errorf("customize.sh dirty-tree failure output %q does not mention the dirty tree", output)
	}
}

func assertOriginAbsent(t *testing.T, root, origin string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" && filepath.Base(path) != "go.mod" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), origin) {
			t.Errorf("%s still references the old origin %s", path, origin)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk customized copy: %v", err)
	}
}

func copyRepository(t *testing.T, destination string) {
	t.Helper()
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == "." {
			return nil
		}
		if path == ".git" || path == "dist" || path == "coverage.out" || path == "tmp" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(destination, path)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o755)
	})
	if err != nil {
		t.Fatalf("copy repository: %v", err)
	}
}

func commitRepository(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "bundle-test@example.com")
	runGit(t, dir, "config", "user.name", "Bundle Test")
	runGit(t, dir, "config", "commit.gpgsign", "false")
	runGit(t, dir, "config", "tag.gpgsign", "false")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "--no-gpg-sign", "--no-verify", "-m", "bundle test snapshot")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func repositoryEnv(origin string) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "GITHUB_REPOSITORY=") {
			continue
		}
		env = append(env, entry)
	}
	return append(env, "GITHUB_REPOSITORY="+strings.TrimPrefix(origin, "github.com/"))
}

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const templateOrigin = "github.com/digitalygo/smidja-bundle-template"

func TestCustomizeScriptRewritesIdentity(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain is required to verify the customization build")
	}
	workDir := t.TempDir()
	copyRepository(t, workDir)
	newOrigin := "github.com/acme/cool-bundle"

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
	if strings.Contains(string(bundleGo), templateOrigin) {
		t.Errorf("customized bundle.go still references the template origin:\n%s", bundleGo)
	}

	mainGo, err := os.ReadFile(filepath.Join(workDir, "main.go"))
	if err != nil {
		t.Fatalf("read customized main.go: %v", err)
	}
	if strings.Contains(string(mainGo), templateOrigin) {
		t.Errorf("customized main.go still imports the template module path:\n%s", mainGo)
	}
}

func TestCustomizeScriptRejectsTemplateOriginAndInvalidPaths(t *testing.T) {
	workDir := t.TempDir()
	copyRepository(t, workDir)
	tests := []struct {
		name string
		arg  string
	}{
		{name: "unchanged template origin", arg: templateOrigin},
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

func copyRepository(t *testing.T, destination string) {
	t.Helper()
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == "." {
			return nil
		}
		if path == ".git" || path == "dist" || path == "coverage.out" {
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

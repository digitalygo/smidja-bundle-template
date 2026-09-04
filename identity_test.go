package main

import (
	"os"
	"strings"
	"testing"
)

func TestValidateIdentity(t *testing.T) {
	pseudoVersion := "v0.2.1-0.20260904203549-a8a32dc552af"
	tests := []struct {
		name             string
		origin           string
		modulePath       string
		harnessVersion   string
		minimumHarness   string
		expectedOrigin   string
		wantErrorMessage string
	}{
		{
			name:           "template identity with pseudo-version pin",
			origin:         "github.com/digitalygo/smidja-bundle-template",
			modulePath:     "github.com/digitalygo/smidja-bundle-template",
			harnessVersion: pseudoVersion,
			minimumHarness: "v0.2.1",
		},
		{
			name:           "customized identity with tagged pin",
			origin:         "github.com/acme/cool-bundle",
			modulePath:     "github.com/acme/cool-bundle",
			harnessVersion: "v0.2.0",
			minimumHarness: "v0.2.0",
		},
		{
			name:             "origin missing repository",
			origin:           "github.com/acme",
			modulePath:       "github.com/acme",
			harnessVersion:   pseudoVersion,
			minimumHarness:   "v0.2.1",
			wantErrorMessage: "must be github.com/owner/repo",
		},
		{
			name:             "origin not on github",
			origin:           "gitlab.com/acme/cool-bundle",
			modulePath:       "gitlab.com/acme/cool-bundle",
			harnessVersion:   pseudoVersion,
			minimumHarness:   "v0.2.1",
			wantErrorMessage: "must be github.com/owner/repo",
		},
		{
			name:             "module path does not match origin",
			origin:           "github.com/acme/cool-bundle",
			modulePath:       "github.com/digitalygo/smidja-bundle-template",
			harnessVersion:   pseudoVersion,
			minimumHarness:   "v0.2.1",
			wantErrorMessage: "must match bundle origin",
		},
		{
			name:             "harness pin is a branch name",
			origin:           "github.com/acme/cool-bundle",
			modulePath:       "github.com/acme/cool-bundle",
			harnessVersion:   "alpha",
			minimumHarness:   "v0.2.1",
			wantErrorMessage: "never a branch name",
		},
		{
			name:             "harness pin is a bare main",
			origin:           "github.com/acme/cool-bundle",
			modulePath:       "github.com/acme/cool-bundle",
			harnessVersion:   "main",
			minimumHarness:   "v0.2.1",
			wantErrorMessage: "never a branch name",
		},
		{
			name:             "harness pin lacks the version prefix",
			origin:           "github.com/acme/cool-bundle",
			modulePath:       "github.com/acme/cool-bundle",
			harnessVersion:   "0.2.1",
			minimumHarness:   "v0.2.1",
			wantErrorMessage: "never a branch name",
		},
		{
			name:             "minimum harness is not canonical",
			origin:           "github.com/acme/cool-bundle",
			modulePath:       "github.com/acme/cool-bundle",
			harnessVersion:   pseudoVersion,
			minimumHarness:   "alpha",
			wantErrorMessage: "must be canonical",
		},
		{
			name:             "generated repository still carries the template origin",
			origin:           "github.com/digitalygo/smidja-bundle-template",
			modulePath:       "github.com/digitalygo/smidja-bundle-template",
			harnessVersion:   pseudoVersion,
			minimumHarness:   "v0.2.1",
			expectedOrigin:   "github.com/acme/cool-bundle",
			wantErrorMessage: "run scripts/customize.sh github.com/acme/cool-bundle",
		},
		{
			name:           "generated repository after customization",
			origin:         "github.com/acme/cool-bundle",
			modulePath:     "github.com/acme/cool-bundle",
			harnessVersion: pseudoVersion,
			minimumHarness: "v0.2.1",
			expectedOrigin: "github.com/acme/cool-bundle",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIdentity(tt.origin, tt.modulePath, tt.harnessVersion, tt.minimumHarness, tt.expectedOrigin)
			if tt.wantErrorMessage == "" {
				if err != nil {
					t.Fatalf("validateIdentity() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validateIdentity() = nil, want error containing %q", tt.wantErrorMessage)
			}
			if !strings.Contains(err.Error(), tt.wantErrorMessage) {
				t.Errorf("validateIdentity() = %q, want error containing %q", err, tt.wantErrorMessage)
			}
		})
	}
}

func TestReadGoModIdentity(t *testing.T) {
	tests := []struct {
		name           string
		goModText      string
		wantModule     string
		wantVersion    string
		wantError      bool
		wantErrorParts []string
	}{
		{
			name: "single line require",
			goModText: `module github.com/acme/cool-bundle

go 1.26

require github.com/digitalygo/smidja v0.2.1-0.20260904203549-a8a32dc552af
`,
			wantModule:  "github.com/acme/cool-bundle",
			wantVersion: "v0.2.1-0.20260904203549-a8a32dc552af",
		},
		{
			name: "require block",
			goModText: `module github.com/acme/cool-bundle

go 1.26

require (
	github.com/digitalygo/smidja v0.2.0
)
`,
			wantModule:  "github.com/acme/cool-bundle",
			wantVersion: "v0.2.0",
		},
		{
			name: "missing module line",
			goModText: `go 1.26

require github.com/digitalygo/smidja v0.2.0
`,
			wantError:      true,
			wantErrorParts: []string{"no module line"},
		},
		{
			name: "missing harness require",
			goModText: `module github.com/acme/cool-bundle

go 1.26
`,
			wantError:      true,
			wantErrorParts: []string{"no github.com/digitalygo/smidja require line"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modulePath, harnessVersion, err := readGoModIdentity(tt.goModText)
			if tt.wantError {
				if err == nil {
					t.Fatalf("readGoModIdentity() = %q, %q, want error", modulePath, harnessVersion)
				}
				for _, part := range tt.wantErrorParts {
					if !strings.Contains(err.Error(), part) {
						t.Errorf("readGoModIdentity() error = %q, want it to contain %q", err, part)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("readGoModIdentity() = %v, want nil", err)
			}
			if modulePath != tt.wantModule {
				t.Errorf("readGoModIdentity() module = %q, want %q", modulePath, tt.wantModule)
			}
			if harnessVersion != tt.wantVersion {
				t.Errorf("readGoModIdentity() harness version = %q, want %q", harnessVersion, tt.wantVersion)
			}
		})
	}
}

func TestRepositoryIdentityMatchesGoMod(t *testing.T) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	modulePath, harnessVersion, err := readGoModIdentity(string(data))
	if err != nil {
		t.Fatalf("parse go.mod: %v", err)
	}
	if err := validateIdentity(bundleOrigin, modulePath, harnessVersion, minimumHarness, ""); err != nil {
		t.Errorf("repository identity is invalid: %v", err)
	}
}

func TestMinimumHarnessMatchesPinnedHarnessBase(t *testing.T) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	_, harnessVersion, err := readGoModIdentity(string(data))
	if err != nil {
		t.Fatalf("parse go.mod: %v", err)
	}
	base := strings.SplitN(harnessVersion, "-", 2)[0]
	if minimumHarness != base {
		t.Errorf("minimumHarness %q does not match the base version %q of the pinned harness %q", minimumHarness, base, harnessVersion)
	}
}

func TestBundleOriginMatchesGitHubRepository(t *testing.T) {
	repository := os.Getenv("GITHUB_REPOSITORY")
	if repository == "" {
		t.Skip("GITHUB_REPOSITORY is only set by GitHub Actions")
	}
	want := "github.com/" + repository
	if bundleOrigin != want {
		t.Errorf("bundle origin %q does not match this repository %q: a repository generated from the template must run scripts/customize.sh %s before its CI can pass", bundleOrigin, want, want)
	}
}

package main

import (
	"fmt"
	"regexp"
	"strings"
)

const harnessModulePath = "github.com/digitalygo/smidja"

var (
	githubOriginPattern     = regexp.MustCompile(`^github\.com/[A-Za-z0-9][A-Za-z0-9-]{0,38}/[A-Za-z0-9._-]+$`)
	immutableVersionPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+(-0\.\d{14}-[0-9a-f]{12})?$`)
	canonicalVersionPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
)

func validateIdentity(origin, modulePath, harnessVersion, minimumHarnessVersion, expectedOrigin string) error {
	if !githubOriginPattern.MatchString(origin) {
		return fmt.Errorf("bundle origin %q must be github.com/owner/repo", origin)
	}
	if modulePath != origin {
		return fmt.Errorf("go.mod module %q must match bundle origin %q", modulePath, origin)
	}
	if !immutableVersionPattern.MatchString(harnessVersion) {
		return fmt.Errorf("harness pin %q must be an immutable go module version such as v1.2.3 or a pseudo-version, never a branch name", harnessVersion)
	}
	if !canonicalVersionPattern.MatchString(minimumHarnessVersion) {
		return fmt.Errorf("minimum harness version %q must be canonical vMAJOR.MINOR.PATCH", minimumHarnessVersion)
	}
	if expectedOrigin != "" && origin != expectedOrigin {
		return fmt.Errorf("bundle origin %q does not match this repository %q: run scripts/customize.sh %s to rename the bundle before releasing", origin, expectedOrigin, expectedOrigin)
	}
	return nil
}

func readGoModIdentity(goModText string) (modulePath string, harnessVersion string, err error) {
	inRequireBlock := false
	foundModule := false
	for _, line := range strings.Split(goModText, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "require (":
			inRequireBlock = true
			continue
		case trimmed == ")" && inRequireBlock:
			inRequireBlock = false
			continue
		case trimmed == "" || strings.HasPrefix(trimmed, "//"):
			continue
		}
		fields := strings.Fields(trimmed)
		switch {
		case !foundModule && fields[0] == "module" && len(fields) >= 2:
			modulePath = fields[1]
			foundModule = true
		case fields[0] == "require" && len(fields) >= 3 && fields[1] == harnessModulePath:
			harnessVersion = fields[2]
		case inRequireBlock && fields[0] == harnessModulePath && len(fields) >= 2:
			harnessVersion = fields[1]
		}
	}
	if !foundModule {
		return "", "", fmt.Errorf("go.mod has no module line")
	}
	if harnessVersion == "" {
		return "", "", fmt.Errorf("go.mod has no %s require line", harnessModulePath)
	}
	return modulePath, harnessVersion, nil
}

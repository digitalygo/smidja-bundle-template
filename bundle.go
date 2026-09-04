package main

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/digitalygo/smidja-bundle-template/extensions/example"
	"github.com/digitalygo/smidja/sdk"
)

var (
	bundleOrigin  = "github.com/digitalygo/smidja-bundle-template"
	bundleVersion = "dev"
	bundleCommit  = "none"
)

const minimumHarness = "v0.2.1"

var defaultsEnvPath = "config/defaults.env"

//go:embed content
var contentFS embed.FS

func Bundle() sdk.Bundle {
	return sdk.Bundle{
		ID:             bundleID(),
		Origin:         bundleOrigin,
		FS:             contentRoot(),
		Extensions:     []sdk.Extension{example.New()},
		ConfigDefaults: defaultsMap(),
		MinimumHarness: minimumHarness,
	}
}

func BuildInfo() sdk.BuildInfo {
	return sdk.BuildInfo{
		Origin:  bundleOrigin,
		Version: bundleVersion,
		Commit:  bundleCommit,
	}
}

func bundleID() string {
	return path.Base(bundleOrigin)
}

func contentRoot() fs.FS {
	sub, err := fs.Sub(contentFS, "content")
	if err != nil {
		panic(fmt.Sprintf("bundle: embed content root: %v", err))
	}
	return sub
}

func defaultsMap() map[string]any {
	parsed := parseDefaults()
	if len(parsed) == 0 {
		return nil
	}
	out := make(map[string]any, len(parsed))
	for key, value := range parsed {
		out[key] = value
	}
	return out
}

func parseDefaults() map[string]string {
	data, err := fs.ReadFile(contentRoot(), defaultsEnvPath)
	if err != nil {
		return map[string]string{}
	}
	return parseDefaultsText(string(data))
}

func parseDefaultsText(text string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out
}

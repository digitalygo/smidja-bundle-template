package main

import (
	"io/fs"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/digitalygo/smidja/sdk"
)

func TestBundleIdentity(t *testing.T) {
	b := Bundle()
	if b.ID == "" {
		t.Fatal("bundle ID is empty")
	}
	if b.ID != path.Base(bundleOrigin) {
		t.Errorf("bundle ID %q is not the repository name of origin %q", b.ID, bundleOrigin)
	}
	if b.Origin != bundleOrigin {
		t.Errorf("bundle origin %q does not match the build origin variable %q", b.Origin, bundleOrigin)
	}
	if b.MinimumHarness == "" {
		t.Error("bundle minimum harness is empty")
	}
}

func TestBundleOriginIsValidGitHubRepository(t *testing.T) {
	b := Bundle()
	parts := strings.Split(b.Origin, "/")
	if len(parts) != 3 || parts[0] != "github.com" || parts[1] == "" || parts[2] == "" {
		t.Fatalf("bundle origin %q must be github.com/owner/repo", b.Origin)
	}
}

func TestBundleComposesEmbeddedContentAndExtensions(t *testing.T) {
	b := Bundle()
	if b.FS == nil {
		t.Fatal("bundle filesystem is nil")
	}
	for _, required := range []string{"skills/example.md", "prompts/example.md", "config/defaults.env"} {
		if _, err := fs.Stat(b.FS, required); err != nil {
			t.Errorf("embedded content %s is missing: %v", required, err)
		}
	}
	if len(b.Extensions) == 0 {
		t.Fatal("bundle registers no extensions")
	}
	seen := map[string]bool{}
	for _, ext := range b.Extensions {
		id := ext.ID()
		if id == "" {
			t.Error("extension has an empty ID")
		}
		if seen[id] {
			t.Errorf("duplicate extension ID %q", id)
		}
		seen[id] = true
	}
	if _, ok := b.Extensions[0].(sdk.Extension); !ok {
		t.Error("bundle extension does not satisfy sdk.Extension")
	}
}

func TestParseDefaultsText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want map[string]string
	}{
		{
			name: "key value pairs",
			text: "SMIDJA_EXEC_TIMEOUT_SECS=30\n",
			want: map[string]string{"SMIDJA_EXEC_TIMEOUT_SECS": "30"},
		},
		{
			name: "blank lines and comments are skipped",
			text: "# a comment\n\nSMIDJA_MAX_OUTPUT_BYTES=1048576\n   \n",
			want: map[string]string{"SMIDJA_MAX_OUTPUT_BYTES": "1048576"},
		},
		{
			name: "value keeps everything after the first equals sign",
			text: "SMIDJA_MODEL=provider/model-name\n",
			want: map[string]string{"SMIDJA_MODEL": "provider/model-name"},
		},
		{
			name: "lines without an equals sign are skipped",
			text: "BROKEN LINE\nSMIDJA_RETRY=on\n",
			want: map[string]string{"SMIDJA_RETRY": "on"},
		},
		{
			name: "surrounding whitespace is trimmed",
			text: "  SMIDJA_EXEC_TIMEOUT_SECS = 45  \n",
			want: map[string]string{"SMIDJA_EXEC_TIMEOUT_SECS": "45"},
		},
		{
			name: "empty text",
			text: "",
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDefaultsText(tt.text)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseDefaultsText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultsMapFromEmbeddedConfig(t *testing.T) {
	got := defaultsMap()
	want := map[string]any{
		"SMIDJA_EXEC_TIMEOUT_SECS": "30",
		"SMIDJA_MAX_OUTPUT_BYTES":  "1048576",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("defaultsMap() = %v, want %v", got, want)
	}
}

func TestParseDefaultsMissingFileYieldsEmptyMap(t *testing.T) {
	saved := defaultsEnvPath
	defaultsEnvPath = "config/does-not-exist.env"
	t.Cleanup(func() { defaultsEnvPath = saved })
	if got := parseDefaults(); len(got) != 0 {
		t.Errorf("parseDefaults() with missing file = %v, want empty", got)
	}
}

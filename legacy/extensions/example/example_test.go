package example

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/digitalygo/smidja/sdk"
)

type recordingAPI struct {
	registeredTools []sdk.Tool
	registerErr     error
}

func (a *recordingAPI) RegisterTool(t sdk.Tool) error {
	if a.registerErr != nil {
		return a.registerErr
	}
	a.registeredTools = append(a.registeredTools, t)
	return nil
}

func (a *recordingAPI) UnregisterTool(name string) error { return nil }

func (a *recordingAPI) ActiveTools() []string { return nil }

func (a *recordingAPI) SetActiveTools(names []string) error { return nil }

func (a *recordingAPI) AllTools() []sdk.ToolInfo { return nil }

func (a *recordingAPI) ConfigValue(key string) string { return "" }

func (a *recordingAPI) RegisterCommand(name string, cmd sdk.Command) error { return nil }

func (a *recordingAPI) Commands() []sdk.CommandInfo { return nil }

func (a *recordingAPI) SendMessage(msg sdk.CustomMessage, opts sdk.SendOptions) error { return nil }

func (a *recordingAPI) SendUserMessage(text string, opts sdk.SendOptions) error { return nil }

func (a *recordingAPI) AppendEntry(customType string, data any) error { return nil }

func (a *recordingAPI) SetSessionName(name string) error { return nil }

func (a *recordingAPI) LabelEntry(entryID, label string) error { return nil }

func (a *recordingAPI) SetModel(m sdk.Model) error { return nil }

func (a *recordingAPI) SetThinkingLevel(level sdk.ThinkingLevel) error { return nil }

func (a *recordingAPI) RegisterProvider(name string, cfg sdk.ProviderConfig) error { return nil }

func (a *recordingAPI) RemoveProvider(name string) error { return nil }

func (a *recordingAPI) RegisterFlag(name string, opts sdk.FlagOptions) error { return nil }

func (a *recordingAPI) Flags() map[string]any { return nil }

func (a *recordingAPI) Exec(command string, args []string, opts sdk.ExecOptions) (*sdk.ExecResult, error) {
	return nil, nil
}

func (a *recordingAPI) EmitCustomEvent(name string, data any) error { return nil }

var _ sdk.API = (*recordingAPI)(nil)

func TestExtensionIdentity(t *testing.T) {
	ext := New()
	if ext.ID() != "example.bundle" {
		t.Errorf("New().ID() = %q, want %q", ext.ID(), "example.bundle")
	}
}

func TestSetupRegistersDigestTool(t *testing.T) {
	api := &recordingAPI{}
	ext := New()
	setup, ok := ext.(sdk.SetupHook)
	if !ok {
		t.Fatal("example extension does not implement sdk.SetupHook")
	}
	if err := setup.Setup(api); err != nil {
		t.Fatalf("Setup() = %v, want nil", err)
	}
	if len(api.registeredTools) != 1 {
		t.Fatalf("Setup() registered %d tools, want 1", len(api.registeredTools))
	}
	if got := api.registeredTools[0].Name(); got != digestToolName {
		t.Errorf("registered tool name = %q, want %q", got, digestToolName)
	}
}

func TestSetupNilAPIFails(t *testing.T) {
	ext := New()
	setup := ext.(sdk.SetupHook)
	err := setup.Setup(nil)
	if err == nil {
		t.Fatal("Setup(nil) = nil, want an error")
	}
	if got := err.Error(); got != "example extension: setup requires a non-nil API" {
		t.Errorf("Setup(nil) error = %q, want the nil API diagnostic", got)
	}
}

func TestSetupWrapsRegisterToolFailure(t *testing.T) {
	api := &recordingAPI{registerErr: errors.New("catalog closed")}
	ext := New()
	setup := ext.(sdk.SetupHook)
	err := setup.Setup(api)
	if err == nil {
		t.Fatal("Setup() with a failing register = nil, want an error")
	}
	if !errors.Is(err, api.registerErr) {
		t.Errorf("Setup() error %v does not wrap the register failure %v", err, api.registerErr)
	}
}

func TestDigestToolSurface(t *testing.T) {
	tool := digestTool{}
	if tool.Name() != "bundle_digest" {
		t.Errorf("Name() = %q, want bundle_digest", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("Description() is empty")
	}
	var schema map[string]any
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("Schema() is not valid JSON: %v", err)
	}
	if schema["type"] != "object" {
		t.Errorf("Schema() type = %v, want object", schema["type"])
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("Schema() has no properties object: %v", schema["properties"])
	}
	if _, ok := properties["text"]; !ok {
		t.Error("Schema() has no text property")
	}
	required, ok := schema["required"].([]any)
	if !ok || len(required) != 1 || required[0] != "text" {
		t.Errorf("Schema() required = %v, want [text]", schema["required"])
	}
}

package example

import (
	"context"
	"encoding/json"
	"testing"
)

func TestDigestToolExec(t *testing.T) {
	tests := []struct {
		name      string
		args      string
		wantText  string
		wantError bool
	}{
		{
			name:     "ascii text",
			args:     `{"text":"smidja"}`,
			wantText: "f5071d1b31e55be00b61b18975d579e443898b1d8046eba23a61584f30e1167d",
		},
		{
			name:     "empty text hashes to the empty digest",
			args:     `{"text":""}`,
			wantText: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "unicode text",
			args:     `{"text":"héllo ✦"}`,
			wantText: "fb81dadc9a6de7b137241b20242c4430f1af97e188c0a19ca2401743b9274ce9",
		},
		{
			name:     "unknown fields are ignored",
			args:     `{"text":"smidja","extra":true}`,
			wantText: "f5071d1b31e55be00b61b18975d579e443898b1d8046eba23a61584f30e1167d",
		},
		{
			name:      "missing text argument",
			args:      `{}`,
			wantError: true,
		},
		{
			name:      "null text argument",
			args:      `{"text":null}`,
			wantError: true,
		},
		{
			name:      "malformed json",
			args:      `{"text":`,
			wantError: true,
		},
		{
			name:      "json array instead of object",
			args:      `["text"]`,
			wantError: true,
		},
	}
	tool := digestTool{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.Exec(context.Background(), json.RawMessage(tt.args))
			if tt.wantError {
				if !result.IsError {
					t.Fatalf("Exec(%s) = %+v, want an error result", tt.args, result)
				}
				if len(result.Content) != 1 || result.Content[0].Type != "text" || result.Content[0].Text == "" {
					t.Errorf("Exec(%s) error content = %+v, want one non-empty text block", tt.args, result.Content)
				}
				return
			}
			if result.IsError {
				t.Fatalf("Exec(%s) reported an error: %+v", tt.args, result.Content)
			}
			if len(result.Content) != 1 || result.Content[0].Type != "text" {
				t.Fatalf("Exec(%s) content = %+v, want one text block", tt.args, result.Content)
			}
			if result.Content[0].Text != tt.wantText {
				t.Errorf("Exec(%s) digest = %q, want %q", tt.args, result.Content[0].Text, tt.wantText)
			}
		})
	}
}

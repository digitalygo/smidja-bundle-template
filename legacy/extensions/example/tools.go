package example

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/digitalygo/smidja/sdk"
)

const digestToolName = "bundle_digest"

type digestArgs struct {
	Text *string `json:"text"`
}

type digestTool struct{}

func (digestTool) Name() string {
	return digestToolName
}

func (digestTool) Description() string {
	return "Compute the SHA-256 hex digest of the given text. Deterministic, offline, and credential free."
}

func (digestTool) Schema() json.RawMessage {
	schema, err := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text": map[string]any{
				"type":        "string",
				"description": "Text to hash",
			},
		},
		"required": []string{"text"},
	})
	if err != nil {
		return nil
	}
	return schema
}

func (digestTool) Exec(ctx context.Context, args json.RawMessage) sdk.Result {
	var parsed digestArgs
	if err := json.Unmarshal(args, &parsed); err != nil {
		return errorResult(fmt.Sprintf("%s: decode arguments: %v", digestToolName, err))
	}
	if parsed.Text == nil {
		return errorResult(fmt.Sprintf("%s: missing required argument text", digestToolName))
	}
	sum := sha256.Sum256([]byte(*parsed.Text))
	return sdk.Result{
		Content: []sdk.Block{{Type: "text", Text: hex.EncodeToString(sum[:])}},
	}
}

func errorResult(message string) sdk.Result {
	return sdk.Result{
		Content: []sdk.Block{{Type: "text", Text: message}},
		IsError: true,
	}
}

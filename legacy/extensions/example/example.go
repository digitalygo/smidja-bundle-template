package example

import (
	"errors"
	"fmt"

	"github.com/digitalygo/smidja/sdk"
)

const extensionID = "example.bundle"

type Extension struct{}

func New() sdk.Extension {
	return Extension{}
}

func (Extension) ID() string {
	return extensionID
}

func (Extension) Setup(api sdk.API) error {
	if api == nil {
		return errors.New("example extension: setup requires a non-nil API")
	}
	if err := api.RegisterTool(digestTool{}); err != nil {
		return fmt.Errorf("example extension: register %s: %w", digestToolName, err)
	}
	return nil
}

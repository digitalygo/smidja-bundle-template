package main

import (
	"context"
	"os"

	"github.com/digitalygo/smidja"
)

func main() {
	os.Exit(smidja.Run(context.Background(), Bundle(), BuildInfo(), os.Args[1:]))
}

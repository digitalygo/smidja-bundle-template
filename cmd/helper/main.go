package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func run(input io.Reader, output io.Writer) error {
	var args struct {
		Text string `json:"text"`
	}
	decoder := json.NewDecoder(io.LimitReader(input, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&args); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("expected one JSON object")
	}
	if strings.TrimSpace(args.Text) == "" {
		return fmt.Errorf("text must not be empty")
	}
	return json.NewEncoder(output).Encode(map[string]string{"text": args.Text})
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

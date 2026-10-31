package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }
func TestRun(t *testing.T) {
	var output bytes.Buffer
	if err := run(strings.NewReader(`{"text":"hello"}`), &output); err != nil {
		t.Fatal(err)
	}
	var value map[string]string
	if err := json.Unmarshal(output.Bytes(), &value); err != nil || value["text"] != "hello" {
		t.Fatal(output.String(), err)
	}
	for _, raw := range []string{`{`, `{}`, `{"text":" "}`, `{"text":1}`, `{"text":"a","unknown":1}`, `{"text":"a"}{}`, `{"text":"a"} junk`} {
		if err := run(strings.NewReader(raw), &output); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	if err := run(strings.NewReader(`{"text":"a"}`), failedWriter{}); err == nil {
		t.Fatal("writer failure lost")
	}
}

func TestMainSuccess(t *testing.T) {
	input, err := os.CreateTemp(t.TempDir(), "input")
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	if _, err = input.WriteString("{\"text\":\"main\"}"); err != nil {
		t.Fatal(err)
	}
	if _, err = input.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	output, err := os.CreateTemp(t.TempDir(), "output")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	originalInput, originalOutput := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = input, output
	defer func() { os.Stdin, os.Stdout = originalInput, originalOutput }()
	main()
}

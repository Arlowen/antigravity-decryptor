package test

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestCLIRejectsExtraListArguments(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/antigravity-decryptor", "list", "extra")
	cmd.Dir = ".."

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected CLI command to fail for extra list arguments")
	}

	output := stderr.String()
	if !strings.Contains(output, "too many arguments for list") {
		t.Fatalf("unexpected stderr: %s", output)
	}
}

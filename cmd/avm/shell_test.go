package main

import (
	"errors"
	"os"
	"testing"

	"github.com/xz1220/agent-vm/internal/config"
)

func TestShellInitSnippets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tests := []struct {
		shell string
		want  string
	}{
		{shell: "bash", want: bashShellInitSnippet},
		{shell: "zsh", want: zshShellInitSnippet},
		{shell: "fish", want: fishShellInitSnippet},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			out, err := executeCommand("shell", "init", tt.shell)
			if err != nil {
				t.Fatalf("shell init returned error: %v", err)
			}
			if out != tt.want {
				t.Fatalf("unexpected shell snippet:\n got: %q\nwant: %q", out, tt.want)
			}
		})
	}

	if _, err := os.Stat(config.AvmDir()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("shell init should not create AVM home, stat err: %v", err)
	}
}

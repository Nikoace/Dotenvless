package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpAndVersion(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"help", []string{"--help"}, "Usage: dvl"},
		{"short help", []string{"-h"}, "Usage: dvl"},
		{"version", []string{"--version"}, "dvl " + Version},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, err bytes.Buffer
			if code := Run(tc.args, nil, &out, &err); code != 0 {
				t.Fatalf("exit = %d", code)
			}
			if !strings.Contains(out.String(), tc.want) || err.Len() != 0 {
				t.Fatalf("unexpected output: %q / %q", out.String(), err.String())
			}
		})
	}
}

func TestInvalidArgumentsDoNotEchoInput(t *testing.T) {
	secret := "FAKE-CLI-SECRET-NEVER-ECHO"
	for _, args := range [][]string{nil, {secret}, {"--help", secret}, {"--version", secret}, {"set", "TOKEN", secret}} {
		var out, err bytes.Buffer
		if code := Run(args, nil, &out, &err); code != 2 {
			t.Errorf("invalid arguments exit = %d, want 2", code)
		}
		if strings.Contains(out.String()+err.String(), secret) {
			t.Fatal("argument leaked")
		}
	}
}

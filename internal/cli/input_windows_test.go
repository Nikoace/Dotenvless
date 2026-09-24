package cli

import (
	"fmt"
	"golang.org/x/term"
	"os"
	"reflect"
	"testing"
)

func TestInteractiveHiddenInput(t *testing.T) {
	mode := os.Getenv("DVL_INTERACTIVE_FIXTURE")
	if mode == "" {
		t.Skip("requires an attached test terminal")
	}
	before, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal("stdin is not a console")
	}
	fmt.Fprintln(os.Stderr, "INTERACTIVE_INPUT_READY")
	value, readErr := readHidden(os.Stdin)
	defer clear(value)
	after, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("terminal mode not restored")
	}
	if mode == "cancel" {
		if readErr == nil || len(value) != 0 {
			t.Fatal("cancelled input was accepted")
		}
	} else if readErr != nil || string(value) != "FAKE_TTY_SECRET_2468" {
		t.Fatal("hidden input mismatch")
	}
	fmt.Fprintln(os.Stderr, "INTERACTIVE_INPUT_OK")
}

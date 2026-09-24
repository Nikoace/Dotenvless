package runner

import (
	"fmt"
	"os"
	"os/signal"
	"testing"
)

func TestInteractiveCtrlC(t *testing.T) {
	if os.Getenv("DVL_INTERACTIVE_RUN") != "1" {
		t.Skip("requires attached console")
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVL_CTRL_CHILD", "1")
	code, err := Run([]string{binary, "-test.run=^TestCtrlChild$"}, nil, os.Stdin, os.Stdout, os.Stderr)
	if err != nil || code != 29 {
		t.Fatalf("Ctrl+C exit not preserved: %d %v", code, err)
	}
	fmt.Fprintln(os.Stdout, "CTRL_C_OK")
}
func TestCtrlChild(t *testing.T) {
	if os.Getenv("DVL_CTRL_CHILD") != "1" {
		return
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	fmt.Fprintln(os.Stdout, "CTRL_C_READY")
	<-signals
	os.Exit(29)
}

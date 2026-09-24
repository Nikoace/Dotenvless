package cli

import (
	"fmt"
	"io"
	"os"
)

var Version = "0.1.0-dev"

const help = `Dotenvless - local secrets for Windows

Usage: dvl <command>

Options:
  --help, -h    Show help
  --version     Show version

Project and secret commands will be added in the next milestones.
`

func Run(args []string, stdin *os.File, stdout, stderr io.Writer) int {
	if len(args) == 1 {
		switch args[0] {
		case "--help", "-h":
			fmt.Fprint(stdout, help)
			return 0
		case "--version":
			fmt.Fprintln(stdout, "dvl "+Version)
			return 0
		}
	}
	// Never echo rejected arguments: they may contain a secret.
	fmt.Fprintln(stderr, "Invalid arguments. Use dvl --help.")
	return 2
}

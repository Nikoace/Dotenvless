package cli

import (
	"dotenvless/internal/config"
	"dotenvless/internal/crypto"
	"dotenvless/internal/project"
	"dotenvless/internal/runner"
	"dotenvless/internal/vault"
	"fmt"
	"io"
	"os"
)

var Version = "0.1.0-dev"

const help = `Dotenvless - local secrets for Windows

Usage: dvl <command>

Commands:
  init         Initialize this Git project
  set <KEY>    Read a secret from a hidden terminal prompt
  list         List secret names
  unset <KEY>  Remove a secret
  status       Show Git project identity
  run -- <COMMAND> [ARG...]  Run a child with project secrets

Options:
  --help, -h   Show help
  --version    Show version
`

type app struct {
	readSecret func(*os.File) ([]byte, error)
}

func Run(args []string, stdin *os.File, stdout, stderr io.Writer) int {
	return (app{readSecret: readHidden}).run(args, stdin, stdout, stderr)
}
func (a app) run(args []string, stdin *os.File, stdout, stderr io.Writer) int {
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
	valid := len(args) == 1 && (args[0] == "init" || args[0] == "list" || args[0] == "status")
	valid = valid || (len(args) == 2 && (args[0] == "set" || args[0] == "unset"))
	valid = valid || (len(args) >= 3 && args[0] == "run" && args[1] == "--")
	if !valid {
		fmt.Fprintln(stderr, "Invalid arguments. Use dvl --help.")
		return 2
	}
	fail := func(err error) int { fmt.Fprintln(stderr, err.Error()); return 1 }
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, "Cannot read working directory.")
		return 1
	}
	p, err := project.Discover(cwd)
	if err != nil {
		return fail(err)
	}
	if args[0] == "status" {
		fmt.Fprintf(stdout, "Project: %s\nRoot: %s\nID: %s\n", p.Name, p.Root, p.ID)
		return 0
	}
	path, err := config.VaultPath(p.Root)
	if err != nil {
		return fail(err)
	}
	s := vault.New(path, crypto.DPAPI{})
	switch args[0] {
	case "run":
		values, err := s.Values(p.ID)
		if err != nil {
			return fail(err)
		}
		defer clear(values)
		code, err := runner.Run(args[2:], values, stdin, stdout, stderr)
		if err != nil {
			return fail(err)
		}
		return code
	case "init":
		if err := s.Init(p.ID); err != nil {
			return fail(err)
		}
		fmt.Fprintln(stdout, "Initialized "+p.Name)
	case "list":
		keys, err := s.Keys(p.ID)
		if err != nil {
			return fail(err)
		}
		for _, key := range keys {
			fmt.Fprintln(stdout, key)
		}
	case "set", "unset":
		key, err := vault.NormalizeKey(args[1])
		if err != nil {
			return fail(err)
		}
		if args[0] == "unset" {
			if err := s.Unset(p.ID, key); err != nil {
				return fail(err)
			}
			fmt.Fprintln(stdout, "Removed "+key)
			return 0
		}
		if _, err := s.Keys(p.ID); err != nil {
			return fail(err)
		}
		fmt.Fprint(stderr, "Secret: ")
		value, err := a.readSecret(stdin)
		fmt.Fprintln(stderr)
		if err != nil {
			fmt.Fprintln(stderr, "Cannot read secret: cancelled, invalid input, or no interactive terminal.")
			return 1
		}
		defer clear(value)
		if err := s.Set(p.ID, key, value); err != nil {
			return fail(err)
		}
		fmt.Fprintln(stdout, "Saved "+key)
	}
	return 0
}

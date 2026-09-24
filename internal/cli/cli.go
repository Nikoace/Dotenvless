package cli

import (
	"dotenvless/internal/config"
	"dotenvless/internal/crypto"
	"dotenvless/internal/dotenv"
	"dotenvless/internal/project"
	"dotenvless/internal/runner"
	"dotenvless/internal/vault"
	"fmt"
	"io"
	"os"
	"strings"
)

var Version = "0.1.0-dev"

const help = `Dotenvless - local secrets for Windows

Usage: dvl <command>

Commands:
  init         Initialize this Git project
  set <KEY>    Read a secret from a hidden terminal prompt
  list         List secret names
  example      Create a key-only .env.example at the Git root
  import [--overwrite] <FILE>  Import a dotenv file without deleting it
  unset <KEY>  Remove a secret
  status [DIRECTORY]  Inspect the current or specified Git project
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
	valid := len(args) == 1 && (args[0] == "init" || args[0] == "list" || args[0] == "status" || args[0] == "example")
	valid = valid || (len(args) == 2 && (args[0] == "set" || args[0] == "unset"))
	valid = valid || (len(args) == 2 && args[0] == "status" && args[1] != "" && !strings.HasPrefix(args[1], "-"))
	valid = valid || (len(args) >= 3 && args[0] == "run" && args[1] == "--")
	importPath, overwrite, validImport := importArguments(args)
	valid = valid || validImport
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
	start := cwd
	if args[0] == "status" && len(args) == 2 {
		start = args[1]
	}
	p, err := project.Discover(start)
	if err != nil {
		return fail(err)
	}
	if args[0] == "status" {
		return printStatus(p, stdout, stderr)
	}
	path, err := config.VaultPath(p.Root)
	if err != nil {
		return fail(err)
	}
	s := vault.New(path, crypto.DPAPI{})
	switch args[0] {
	case "example":
		keys, err := s.Keys(p.ID)
		if err != nil {
			return fail(err)
		}
		if err := writeExample(p.Root, keys); err != nil {
			return fail(err)
		}
		fmt.Fprintln(stdout, "Created .env.example")
		return 0
	case "import":
		if _, err := s.Keys(p.ID); err != nil {
			return fail(err)
		}
		file, err := os.Open(importPath)
		if err != nil {
			fmt.Fprintln(stderr, "Cannot open dotenv source.")
			return 1
		}
		values, err := dotenv.Parse(file)
		_ = file.Close()
		if err != nil {
			return fail(err)
		}
		defer func() {
			for _, value := range values {
				clear(value)
			}
		}()
		keys, err := s.Import(p.ID, values, overwrite)
		if err != nil {
			return fail(err)
		}
		fmt.Fprintf(stdout, "Found %d variables\n", len(keys))
		for _, key := range keys {
			fmt.Fprintln(stdout, key+" imported")
		}
		return 0
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

package cli

import (
	"dotenvless/internal/application"
	"dotenvless/internal/config"
	"dotenvless/internal/crypto"
	"dotenvless/internal/runner"
	"dotenvless/internal/vault"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var Version = "0.2.0-dev"

const help = `Dotenvless - local environment management for Windows

Usage: dvl [--env NAME | --base] <command>

Commands:
  init         Register the Git project or an ordinary directory
  set <KEY>    Read a value from a hidden terminal prompt
  list         List effective variable names (no values)
  example      Create a key-only .env.example at the project root
  import [--overwrite] <FILE>  Import dotenv without deleting its source
  unset <KEY>  Remove a binding from the selected layer
  status [DIRECTORY]  Inspect the current or specified project
  run -- <COMMAND> [ARG...]  Run with the selected environment
  migrate      Explicitly upgrade a v1 Vault with an encrypted backup

Selectors:
  --env NAME   Select an environment (default: local)
  --base       Edit the base layer; cannot be used with run
  The desktop client's selected environment never changes CLI defaults.

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
	invalid := func() int { fmt.Fprintln(stderr, "Invalid arguments. Use dvl --help."); return 2 }
	scope := "local"
	selected := false
	if len(args) > 0 && args[0] == "--base" {
		scope = "base"
		args = args[1:]
		selected = true
	} else if len(args) > 0 && args[0] == "--env" {
		if len(args) < 3 || !vault.ValidEnvironment(args[1]) {
			return invalid()
		}
		scope = args[1]
		args = args[2:]
		selected = true
	}
	valid := len(args) == 1 && (args[0] == "init" || args[0] == "list" || args[0] == "status" || args[0] == "example" || args[0] == "migrate")
	valid = valid || (len(args) == 2 && (args[0] == "set" || args[0] == "unset"))
	valid = valid || (len(args) == 2 && args[0] == "status" && args[1] != "" && !strings.HasPrefix(args[1], "-"))
	valid = valid || (len(args) >= 3 && args[0] == "run" && args[1] == "--" && scope != "base")
	importPath, overwrite, validImport := importArguments(args)
	valid = valid || validImport
	if !valid {
		return invalid()
	}
	if selected && (args[0] == "init" || args[0] == "migrate") {
		return invalid()
	}
	fail := func(e error) int { fmt.Fprintln(stderr, e.Error()); return 1 }
	path, e := config.DefaultVaultPath()
	if e != nil {
		return fail(e)
	}
	service := application.New(path, crypto.DPAPI{})
	if args[0] == "migrate" {
		backup, e := service.Store.Migrate()
		if e != nil {
			return fail(e)
		}
		if backup == "" {
			fmt.Fprintln(stdout, "Vault already uses v2")
		} else {
			fmt.Fprintln(stdout, "Vault upgraded; original encrypted backup retained")
		}
		return 0
	}
	cwd, e := os.Getwd()
	if e != nil {
		return fail(errors.New("cannot read working directory"))
	}
	start := cwd
	if args[0] == "status" && len(args) == 2 {
		start = args[1]
	}
	p, e := service.FindProject(start)
	if args[0] == "init" {
		if e == nil {
			start = p.Root
		}
		p, e = service.Register(start, vault.AnyRevision)
		if e != nil {
			return fail(e)
		}
		fmt.Fprintln(stdout, "Initialized "+p.Name)
		return 0
	}
	if e != nil {
		return fail(e)
	}
	// A Git identity may exist before the registry; also validate storage location.
	if _, e = config.VaultPath(p.Root); e != nil {
		return fail(e)
	}
	if args[0] == "status" {
		return printStatus(p, service, scope, stdout, stderr)
	}
	if args[0] == "set" || args[0] == "unset" || args[0] == "import" {
		state, e := service.State()
		if e != nil {
			return fail(e)
		}
		if state.Version == 1 {
			return fail(vault.ErrMigration)
		}
	}
	switch args[0] {
	case "list", "example":
		entries, _, e := service.Entries(p.ID, scope)
		if e != nil {
			return fail(e)
		}
		keys := []string{}
		for _, entry := range entries {
			if entry.Kind != "disabled" {
				keys = append(keys, entry.Key)
			}
		}
		if args[0] == "example" {
			if e = service.Example(p.ID, scope, p.Root); e != nil {
				return fail(e)
			}
			fmt.Fprintln(stdout, "Created .env.example")
			return 0
		}
		for _, key := range keys {
			fmt.Fprintln(stdout, key)
		}
	case "set", "unset":
		key, e := vault.NormalizeKey(args[1])
		if e != nil {
			return fail(e)
		}
		if args[0] == "unset" {
			if e = service.Unset(p.ID, scope, key, vault.AnyRevision); e != nil {
				return fail(e)
			}
			fmt.Fprintln(stdout, "Removed "+key)
			return 0
		}
		if _, _, e = service.Entries(p.ID, scope); e != nil {
			return fail(e)
		}
		fmt.Fprint(stderr, "Secret: ")
		value, e := a.readSecret(stdin)
		fmt.Fprintln(stderr)
		if e != nil {
			clear(value)
			return fail(errors.New("Cannot read secret: cancelled, invalid input, or no interactive terminal."))
		}
		defer clear(value)
		if e = service.Set(p.ID, scope, key, value, vault.AnyRevision); e != nil {
			return fail(e)
		}
		fmt.Fprintln(stdout, "Saved "+key)
	case "import":
		if _, _, e = service.Entries(p.ID, scope); e != nil {
			return fail(e)
		}
		f, e := os.Open(importPath)
		if e != nil {
			return fail(errors.New("Cannot open dotenv source."))
		}
		preview, e := service.PreviewImport(p.ID, scope, f)
		_ = f.Close()
		if e != nil {
			return fail(e)
		}
		defer preview.Close()
		if e = service.CommitImport(preview, overwrite); e != nil {
			return fail(e)
		}
		fmt.Fprintf(stdout, "Found %d variables\n", len(preview.Keys))
		for _, key := range preview.Keys {
			fmt.Fprintln(stdout, key+" imported")
		}
	case "run":
		resolved, e := service.Resolve(p.ID, scope)
		if e != nil {
			return fail(e)
		}
		defer clear(resolved.Values)
		code, e := runner.RunWithRemoved(args[2:], resolved.Values, resolved.Removed, stdin, stdout, stderr)
		if e != nil {
			return fail(e)
		}
		return code
	}
	return 0
}

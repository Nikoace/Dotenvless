package cli

import (
	"dotenvless/internal/config"
	"dotenvless/internal/crypto"
	"dotenvless/internal/project"
	"dotenvless/internal/vault"
	"dotenvless/internal/workspace"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
)

func printStatus(p project.Identity, stdout, stderr io.Writer) int {
	code := 0
	fmt.Fprintf(stdout, "Project: %s\nRoot: %s\nID: %s\n", p.Name, p.Root, p.ID)
	path, err := config.VaultPath(p.Root)
	if err != nil {
		fmt.Fprintln(stderr, "Vault:", err)
		code = 1
	} else {
		_, err := os.Stat(path)
		switch {
		case errors.Is(err, os.ErrNotExist):
			fmt.Fprintln(stdout, "Vault: not initialized; run dvl init")
		case err != nil:
			fmt.Fprintln(stderr, "Vault: cannot inspect encrypted storage")
			code = 1
		default:
			values, err := vault.New(path, crypto.DPAPI{}).Values(p.ID)
			if errors.Is(err, vault.ErrNotInitialized) {
				fmt.Fprintln(stdout, "Vault: this project is not initialized; run dvl init")
			} else if err != nil {
				fmt.Fprintln(stderr, "Vault:", err)
				code = 1
			} else {
				defer clear(values)
				keys := make([]string, 0, len(values))
				for key := range values {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				if len(keys) == 0 {
					fmt.Fprintln(stdout, "Vault: initialized; no secrets")
				} else {
					fmt.Fprintln(stdout, "Vault: encrypted; Windows DPAPI decryption verified")
					fmt.Fprintln(stdout, "Write scope: Current User")
				}
				fmt.Fprintln(stdout, "Secrets:")
				for _, key := range keys {
					fmt.Fprintln(stdout, "  "+key)
				}
			}
		}
	}
	fmt.Fprintln(stdout, "Workspace scope:", workspace.Scope)
	report, err := workspace.Check(p.Root)
	if err != nil {
		fmt.Fprintln(stderr, "Workspace: scan incomplete")
		return 1
	}
	if len(report.Files) == 0 {
		fmt.Fprintln(stdout, "Environment files: none found in inspected paths")
	} else {
		fmt.Fprintln(stdout, "Environment files detected (contents not inspected):")
		for _, file := range report.Files {
			fmt.Fprintf(stdout, "  %s (Git: %s)\n", file, report.Git[file])
		}
	}
	fmt.Fprintln(stdout, "Git .env:", report.Git[".env"])
	return code
}

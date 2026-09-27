package cli

import (
	"dotenvless/internal/application"
	"dotenvless/internal/project"
	"dotenvless/internal/vault"
	"dotenvless/internal/workspace"
	"errors"
	"fmt"
	"io"
)

func printStatus(p project.Identity, s *application.Service, scope string, stdout, stderr io.Writer) int {
	code := 0
	fmt.Fprintf(stdout, "Project: %s\nRoot: %s\nID: %s\nEnvironment: %s\n", p.Name, p.Root, p.ID, scope)
	entries, _, e := s.Entries(p.ID, scope)
	switch {
	case errors.Is(e, vault.ErrNotInitialized):
		fmt.Fprintln(stdout, "Vault: this project is not initialized; run dvl init")
	case e != nil:
		fmt.Fprintln(stderr, "Vault:", e)
		code = 1
	default:
		active := 0
		for _, entry := range entries {
			if entry.Kind != "disabled" {
				active++
			}
		}
		if e = s.Verify(p.ID, scope); e != nil {
			fmt.Fprintln(stderr, "Vault:", e)
			code = 1
		} else if active == 0 {
			fmt.Fprintln(stdout, "Vault: initialized; no effective secrets")
		} else {
			fmt.Fprintln(stdout, "Vault: encrypted; Windows DPAPI decryption verified")
			fmt.Fprintln(stdout, "Write scope: Current User")
		}
		fmt.Fprintln(stdout, "Secrets:")
		for _, entry := range entries {
			if entry.Kind != "disabled" {
				fmt.Fprintln(stdout, "  "+entry.Key)
			}
		}
	}
	fmt.Fprintln(stdout, "Workspace scope:", workspace.Scope)
	report, e := workspace.Check(p.Root)
	if e != nil {
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

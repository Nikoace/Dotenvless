package workspace

import (
	"context"
	"errors"
	"io/fs"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const Scope = "recursive file-name check; excludes .git, node_modules, .venv, venv, vendor, build, dist, target, .gradle, .cache, .tools, __pycache__; no directory links"

var excluded = map[string]bool{".git": true, "node_modules": true, ".venv": true, "venv": true, "vendor": true, "build": true, "dist": true, "target": true, ".gradle": true, ".cache": true, ".tools": true, "__pycache__": true}

type Report struct {
	Files []string
	Git   map[string]string
}

func Check(root string) (Report, error) {
	report := Report{Files: []string{}, Git: map[string]string{}}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return errors.New("workspace scan incomplete")
		}
		if entry.IsDir() {
			if path == root {
				return nil
			}
			if excluded[strings.ToLower(entry.Name())] {
				return filepath.SkipDir
			}
			linked, err := directoryLink(path)
			if err != nil {
				return errors.New("cannot inspect workspace directory")
			}
			if linked {
				return filepath.SkipDir
			}
			return nil
		}
		name := strings.ToLower(entry.Name())
		if name == ".env.example" || strings.HasPrefix(name, ".env.") && strings.HasSuffix(name, ".example") {
			return nil
		}
		if name == ".env" || strings.HasPrefix(name, ".env.") {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return errors.New("cannot resolve workspace file")
			}
			report.Files = append(report.Files, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return report, err
	}
	sort.Strings(report.Files)
	report.Git[".env"] = gitState(root, ".env")
	for _, file := range report.Files {
		if _, ok := report.Git[file]; !ok {
			report.Git[file] = gitState(root, file)
		}
	}
	return report, nil
}
func gitExit(root string, args ...string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	prefix := []string{"--no-optional-locks", "-c", "core.fsmonitor=false", "-C", root}
	cmd := exec.CommandContext(ctx, "git", append(prefix, args...)...)
	err := cmd.Run()
	if err == nil {
		return 0
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	return -1
}
func gitState(root, path string) string {
	switch gitExit(root, "ls-files", "--error-unmatch", "--", path) {
	case 0:
		return "tracked"
	case 1:
	default:
		return "unknown"
	}
	switch gitExit(root, "check-ignore", "--quiet", "--", path) {
	case 0:
		return "ignored"
	case 1:
		return "not ignored"
	default:
		return "unknown"
	}
}

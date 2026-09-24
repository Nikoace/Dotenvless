package runner

import (
	"errors"
	"golang.org/x/sys/windows"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"unicode"
)

func Run(args []string, secrets map[string]string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	if len(args) == 0 || args[0] == "" {
		return 1, errors.New("command is required")
	}
	command := args[0]
	// Windows repositories commonly contain both a POSIX gradlew and gradlew.bat.
	if strings.EqualFold(filepath.Base(command), "gradlew") {
		if candidate, err := exec.LookPath(command + ".bat"); err == nil {
			command = candidate
		}
	}
	path, err := exec.LookPath(command)
	if err != nil {
		return 1, errors.New("command not found; use an explicit path if it is in the current directory")
	}
	var cmd *exec.Cmd
	extension := strings.ToLower(filepath.Ext(path))
	if extension == ".cmd" || extension == ".bat" {
		if !safeBatch(path, false) {
			return 1, errors.New("batch path contains unsupported shell characters; select a shell explicitly")
		}
		parts := []string{`"` + path + `"`}
		for _, arg := range args[1:] {
			if !safeBatch(arg, true) {
				return 1, errors.New("batch argument cannot be passed safely; select a shell explicitly")
			}
			parts = append(parts, `"`+arg+`"`)
		}
		system, err := windows.GetSystemDirectory()
		if err != nil {
			return 1, errors.New("cannot locate the Windows command processor")
		}
		shell := filepath.Join(system, "cmd.exe")
		cmd = exec.Command(shell)
		cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: syscall.EscapeArg(shell) + ` /d /v:off /s /c "` + strings.Join(parts, " ") + `"`}
	} else {
		cmd = exec.Command(path, args[1:]...)
	}
	cmd.Env = environment(os.Environ(), secrets)
	defer clear(cmd.Env)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	// Console Ctrl+C also reaches the child. Keep the wrapper alive to wait for it.
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)
	err = cmd.Run()
	if err == nil {
		return 0, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), nil
	}
	return 1, errors.New("cannot start or wait for command")
}
func safeBatch(text string, argument bool) bool {
	if strings.ContainsAny(text, "\"%!^&|<>") {
		return false
	}
	for _, r := range text {
		if unicode.IsControl(r) {
			return false
		}
	}
	return !argument || !strings.HasSuffix(text, `\`)
}

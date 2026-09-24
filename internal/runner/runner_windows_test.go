package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func helperEnvironment(t *testing.T, args []string, exit int) {
	t.Helper()
	t.Setenv("DVL_RUN_HELPER", "1")
	encoded, _ := json.Marshal(args)
	t.Setenv("DVL_EXPECT_ARGS", string(encoded))
	t.Setenv("DVL_EXPECT_EXIT", strconv.Itoa(exit))
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVL_EXPECT_CWD", cwd)
	t.Setenv("DVL_PARENT", "inherited")
	t.Setenv("DVL_TOKEN", "parent")
}
func TestDirectProcessEnvironmentArgumentsStreamsAndExit(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"", "space arg", "日本語", "quote\"here", "%PATH%", "&", "tail\\"}
	for _, exit := range []int{0, 37} {
		helperEnvironment(t, args, exit)
		var out, stderr bytes.Buffer
		code, err := Run(append([]string{binary, "-test.run=^TestRunnerHelper$", "--"}, args...), map[string]string{"dvl_token": "FAKE-CHILD-SECRET"}, strings.NewReader("input fixture"), &out, &stderr)
		if err != nil || code != exit {
			t.Fatalf("code=%d error=%v stderr=%s", code, err, stderr.String())
		}
		if out.String() != "child output" || stderr.String() != "child error" {
			t.Fatal("streams changed")
		}
		if os.Getenv("DVL_TOKEN") != "parent" {
			t.Fatal("parent environment changed")
		}
	}
	files, err := os.ReadDir(root)
	if err != nil || len(files) != 0 {
		t.Fatal("runner wrote project files")
	}
}
func TestBatchArgumentsAndGradleWrapperResolution(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "space 日本 (fixture)")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "gradlew.bat")
	body := fmt.Sprintf("@echo off\r\n\"%s\" -test.run=TestRunnerHelper -- %%*\r\nexit /b %%errorlevel%%\r\n", binary)
	if err := os.WriteFile(script, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	args := []string{"", "space arg", "日本語", "(value)", "-Doption=value", "C:\\path"}
	helperEnvironment(t, args, 19)
	var out, stderr bytes.Buffer
	code, err := Run(append([]string{"./gradlew"}, args...), map[string]string{"DVL_TOKEN": "FAKE-CHILD-SECRET"}, strings.NewReader("input fixture"), &out, &stderr)
	if err != nil || code != 19 {
		t.Fatalf("batch code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if out.String() != "child output" || stderr.String() != "child error" {
		t.Fatal("batch streams changed")
	}
	for _, arg := range []string{"%PATH%", "a&b", "a|b", "a>b", "a<b", "a^b", "a!b", "a\"b", "a\nb", "tail\\"} {
		out.Reset()
		stderr.Reset()
		if _, err := Run([]string{script, arg}, nil, nil, &out, &stderr); err == nil {
			t.Fatal("ambiguous batch argument accepted")
		}
		if out.Len() != 0 || stderr.Len() != 0 {
			t.Fatal("rejected command was started")
		}
	}
}
func TestEnvironmentMergeAndMissingCommand(t *testing.T) {
	env := environment([]string{"Path=old", "PATH=last", "KEEP=x", "=C:=C:\\working"}, map[string]string{"path": "new", "TOKEN": "fake"})
	joined := strings.Join(env, "\n")
	if strings.Count(strings.ToUpper(joined), "PATH=") != 1 || !strings.Contains(joined, "PATH=new") || !strings.Contains(joined, "=C:=C:\\working") || !strings.Contains(joined, "KEEP=x") {
		t.Fatalf("bad merge: %v", env)
	}
	for _, args := range [][]string{nil, {"command-that-does-not-exist-dvl-fixture"}} {
		var out, stderr bytes.Buffer
		if code, err := Run(args, nil, nil, &out, &stderr); err == nil || code == 0 {
			t.Fatal("missing command accepted")
		}
	}
}
func TestRunnerHelper(t *testing.T) {
	if os.Getenv("DVL_RUN_HELPER") != "1" {
		return
	}
	marker := -1
	for i, arg := range os.Args {
		if arg == "--" {
			marker = i
			break
		}
	}
	var expected []string
	if json.Unmarshal([]byte(os.Getenv("DVL_EXPECT_ARGS")), &expected) != nil || marker < 0 || !reflect.DeepEqual(os.Args[marker+1:], expected) {
		os.Exit(91)
	}
	cwd, _ := os.Getwd()
	if cwd != os.Getenv("DVL_EXPECT_CWD") || os.Getenv("DVL_TOKEN") != "FAKE-CHILD-SECRET" || os.Getenv("DVL_PARENT") != "inherited" {
		os.Exit(92)
	}
	input, err := io.ReadAll(os.Stdin)
	if err != nil || string(input) != "input fixture" {
		os.Exit(93)
	}
	fmt.Fprint(os.Stdout, "child output")
	fmt.Fprint(os.Stderr, "child error")
	code, _ := strconv.Atoi(os.Getenv("DVL_EXPECT_EXIT"))
	os.Exit(code)
}

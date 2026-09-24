package runner

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstalledToolSmoke(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"Python", []string{"python.exe", "-c", "import os,sys; sys.exit(0 if os.environ['DVL_SMOKE']=='FAKE_TOOL_SECRET' else 42)"}},
		{"Node", []string{"node.exe", "-e", "process.exit(process.env.DVL_SMOKE==='FAKE_TOOL_SECRET'?0:42)"}},
		{"PowerShell", []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "if ($env:DVL_SMOKE -eq 'FAKE_TOOL_SECRET') { exit 0 } else { exit 42 }"}},
		{"CMD", []string{"cmd.exe", "/d", "/s", "/c", "if %DVL_SMOKE%==FAKE_TOOL_SECRET (exit /b 0) else (exit /b 42)"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := exec.LookPath(tc.args[0]); err != nil {
				t.Skip("runtime is not installed")
			}
			var out, stderr bytes.Buffer
			code, err := Run(tc.args, map[string]string{"DVL_SMOKE": "FAKE_TOOL_SECRET"}, nil, &out, &stderr)
			if err != nil || code != 0 {
				t.Fatalf("runtime failed: exit=%d error=%v output=%s", code, err, stderr.String())
			}
			if strings.Contains(out.String()+stderr.String(), "FAKE_TOOL_SECRET") {
				t.Fatal("fixture exposed value")
			}
		})
	}
}
func TestInstalledNpmScript(t *testing.T) {
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm is not installed")
	}
	root := t.TempDir()
	t.Chdir(root)
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"verify":"node verify.cjs"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "verify.cjs"), []byte("process.exit(process.env.DVL_SMOKE==='FAKE_TOOL_SECRET'?0:42)"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code, err := Run([]string{"npm", "run", "verify", "--silent"}, map[string]string{"DVL_SMOKE": "FAKE_TOOL_SECRET", "npm_config_update_notifier": "false", "npm_config_audit": "false", "npm_config_fund": "false"}, nil, &out, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("npm failed: %d %v %s", code, err, stderr.String())
	}
}
func TestInstalledGradle(t *testing.T) {
	launcher := os.Getenv("DVL_TEST_GRADLE")
	if launcher == "" {
		t.Skip("set DVL_TEST_GRADLE to an installed gradle.bat to run the offline integration")
	}
	root, cache := t.TempDir(), t.TempDir()
	t.Chdir(root)
	build := "tasks.register('verifySecret') { doLast { if (System.getenv('DVL_SMOKE') != 'FAKE_TOOL_SECRET') throw new GradleException('Missing fixture environment') } }"
	if err := os.WriteFile(filepath.Join(root, "build.gradle"), []byte(build), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "settings.gradle"), []byte("rootProject.name = 'dvl-fixture'"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code, err := Run([]string{launcher, "--no-daemon", "--offline", "--console=plain", "verifySecret"}, map[string]string{"DVL_SMOKE": "FAKE_TOOL_SECRET", "GRADLE_USER_HOME": cache}, nil, &out, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("Gradle failed: %d %v %s %s", code, err, out.String(), stderr.String())
	}
	if !strings.Contains(out.String(), "BUILD SUCCESSFUL") {
		t.Fatal("Gradle did not execute fixture")
	}
}

package vault_test

import (
	"bytes"
	"dotenvless/internal/crypto"
	"dotenvless/internal/vault"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

func TestRealDPAPIPersistenceAndConcurrentProcesses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.dat")
	s := vault.New(path, crypto.DPAPI{})
	if err := s.Init("p"); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("p", "TOKEN", []byte("FAKE-PERSISTED-SECRET")); err != nil {
		t.Fatal(err)
	}
	values, err := vault.New(path, crypto.DPAPI{}).Values("p")
	if err != nil || values["TOKEN"] != "FAKE-PERSISTED-SECRET" {
		t.Fatal("real DPAPI persistence failed")
	}
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	failures := make(chan error, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cmd := exec.Command(binary, "-test.run=^TestVaultProcessHelper$")
			cmd.Env = append(os.Environ(), "DVL_TEST_VAULT="+path, fmt.Sprintf("DVL_TEST_KEY=CHILD_%d", i))
			output, err := cmd.CombinedOutput()
			if err != nil {
				failures <- fmt.Errorf("helper failed: %v: %s", err, output)
			}
		}(i)
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	keys, err := s.Keys("p")
	if err != nil || len(keys) != 7 {
		t.Fatalf("cross-process update loss: %d %v", len(keys), err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("FAKE-PERSISTED-SECRET")) || bytes.Contains(data, []byte("FAKE-PROCESS")) {
		t.Fatal("plaintext persisted")
	}
}
func TestVaultProcessHelper(t *testing.T) {
	path := os.Getenv("DVL_TEST_VAULT")
	if path == "" {
		return
	}
	s := vault.New(path, crypto.DPAPI{})
	if err := s.Set("p", os.Getenv("DVL_TEST_KEY"), []byte("FAKE-PROCESS")); err != nil {
		os.Exit(11)
	}
	os.Exit(0)
}

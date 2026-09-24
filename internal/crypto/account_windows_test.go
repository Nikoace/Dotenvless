package crypto

import (
	"encoding/json"
	"golang.org/x/sys/windows"
	"os"
	"testing"
)

type accountFixture struct {
	SID        string `json:"sid"`
	Ciphertext []byte `json:"ciphertext"`
}

func TestDPAPIAcrossAccounts(t *testing.T) {
	mode, path := os.Getenv("DVL_DPAPI_CHECK_MODE"), os.Getenv("DVL_DPAPI_CHECK_FILE")
	if mode == "" {
		t.Skip("manual two-account test; see docs/manual-verification.md")
	}
	if path == "" {
		t.Fatal("DVL_DPAPI_CHECK_FILE is required")
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal("cannot identify test account")
	}
	sid := user.User.Sid.String()
	protector := DPAPI{}
	context := []byte("dotenvless-cross-account-test-v1")
	if mode == "write" {
		ciphertext, err := protector.Protect([]byte("FAKE_CROSS_ACCOUNT_PROBE"), context)
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(accountFixture{SID: sid, Ciphertext: ciphertext})
		if err != nil {
			t.Fatal(err)
		}
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			t.Fatal("cannot create a new fixture file")
		}
		_, writeErr := file.Write(data)
		closeErr := file.Close()
		if writeErr != nil || closeErr != nil {
			t.Fatal("cannot write fixture")
		}
		t.Log("Encrypted fake-value fixture created; cross-account rejection is not yet verified.")
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("cannot read fixture; file permission errors do not prove DPAPI rejection")
	}
	var fixture accountFixture
	if json.Unmarshal(data, &fixture) != nil || fixture.SID == "" || len(fixture.Ciphertext) == 0 {
		t.Fatal("invalid fixture")
	}
	plain, openErr := protector.Unprotect(fixture.Ciphertext, context)
	defer clear(plain)
	switch mode {
	case "read-same":
		if sid != fixture.SID || openErr != nil || string(plain) != "FAKE_CROSS_ACCOUNT_PROBE" {
			t.Fatal("same-account fixture validation failed")
		}
	case "read-other":
		if sid == fixture.SID {
			t.Fatal("a genuinely different Windows account is required")
		}
		if openErr == nil {
			t.Fatal("different account decrypted the fixture")
		}
		t.Log("DPAPI rejection verified under a different Windows SID after successfully reading ciphertext.")
	default:
		t.Fatal("mode must be write, read-same, or read-other")
	}
}

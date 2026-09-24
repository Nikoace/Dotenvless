package crypto

import (
	"bytes"
	"testing"
)

func TestDPAPIRoundTripAndBinding(t *testing.T) {
	p := DPAPI{}
	for _, value := range [][]byte{nil, []byte("FAKE-SECRET_日本語-0123456789"), []byte("first\nsecond")} {
		encrypted, err := p.Protect(value, []byte("project:key"))
		if err != nil {
			t.Fatal(err)
		}
		if len(value) > 0 && bytes.Contains(encrypted, value) {
			t.Fatal("plaintext in ciphertext")
		}
		plain, err := p.Unprotect(encrypted, []byte("project:key"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(plain, value) {
			t.Fatal("round trip mismatch")
		}
		if _, err := p.Unprotect(encrypted, []byte("different:key")); err == nil {
			t.Fatal("wrong context accepted")
		}
		encrypted[len(encrypted)-1] ^= 0xff
		if _, err := p.Unprotect(encrypted, []byte("project:key")); err == nil {
			t.Fatal("tampered data accepted")
		}
	}
	if _, err := p.Unprotect([]byte("not DPAPI"), nil); err == nil {
		t.Fatal("invalid ciphertext accepted")
	}
}
func TestDPAPINondeterministic(t *testing.T) {
	p := DPAPI{}
	a, err := p.Protect([]byte("FAKE-SECRET"), nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := p.Protect([]byte("FAKE-SECRET"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a, b) {
		t.Fatal("encryption reused identical ciphertext")
	}
}

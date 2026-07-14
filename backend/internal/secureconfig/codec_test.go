package secureconfig

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestCodecRoundTripDoesNotPersistPlaintext(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	codec, err := New(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	plain := []byte(`{"app_id":"cli_test","app_secret":"secret-value"}`)
	sealed, err := codec.Seal(plain)
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	if bytes.Contains(sealed, []byte("secret-value")) {
		t.Fatalf("sealed config contains plaintext secret: %q", sealed)
	}

	opened, encrypted, err := codec.Open(sealed)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if !encrypted {
		t.Fatal("Open() encrypted = false, want true")
	}
	if !bytes.Equal(opened, plain) {
		t.Fatalf("Open() = %q, want %q", opened, plain)
	}
}

func TestCodecOpenAcceptsLegacyPlaintext(t *testing.T) {
	codec, err := New("")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	legacy := []byte(`{"app_id":"legacy"}`)

	opened, encrypted, err := codec.Open(legacy)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if encrypted {
		t.Fatal("Open() encrypted = true, want false")
	}
	if !bytes.Equal(opened, legacy) {
		t.Fatalf("Open() = %q, want %q", opened, legacy)
	}
}

func TestCodecRejectsInvalidKey(t *testing.T) {
	if _, err := New("not-a-32-byte-base64-key"); err == nil {
		t.Fatal("New() error = nil, want invalid key error")
	}
}

func TestCodecCannotOpenEncryptedValueWithoutKey(t *testing.T) {
	key := bytes.Repeat([]byte{0x24}, 32)
	writer, err := New(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	sealed, err := writer.Seal([]byte(`{"app_secret":"secret"}`))
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	reader, err := New("")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, _, err := reader.Open(sealed); err == nil {
		t.Fatal("Open() error = nil, want missing key error")
	}
}

package adminapi

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/smices/open-idb/internal/db/generated"
	"github.com/smices/open-idb/internal/secureconfig"
)

func TestFeishuConfigResponseDecryptsStoredConfig(t *testing.T) {
	codec, err := secureconfig.New(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x33}, 32)))
	if err != nil {
		t.Fatalf("secureconfig.New() error = %v", err)
	}
	plain := []byte(`{"app_id":"cli_test","app_secret":"secret-value"}`)
	sealed, err := codec.Seal(plain)
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	service := &ConfigDBService{configCodec: codec}
	got, err := service.feishuIdentitySourceConfigFromGetRow(generated.GetFeishuIdentitySourceConfigRow{
		Config: sealed,
	})
	if err != nil {
		t.Fatalf("feishuIdentitySourceConfigFromGetRow() error = %v", err)
	}
	if !bytes.Equal(got.Config, plain) {
		t.Fatalf("config = %q, want %q", got.Config, plain)
	}
}

func TestFeishuConfigResponseKeepsLegacyPlaintextReadable(t *testing.T) {
	codec, err := secureconfig.New(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x55}, 32)))
	if err != nil {
		t.Fatalf("secureconfig.New() error = %v", err)
	}
	plain := []byte(`{"app_id":"legacy"}`)

	service := &ConfigDBService{configCodec: codec}
	got, err := service.feishuIdentitySourceConfigFromGetRow(generated.GetFeishuIdentitySourceConfigRow{
		Config: plain,
	})
	if err != nil {
		t.Fatalf("feishuIdentitySourceConfigFromGetRow() error = %v", err)
	}
	if !bytes.Equal(got.Config, plain) {
		t.Fatalf("config = %q, want %q", got.Config, plain)
	}
}

func TestValidateFeishuConfigRejectsBlankWebhookCredentials(t *testing.T) {
	err := validateFeishuIdentitySourceConfig(false, []byte(`{"verification_token":"   "}`))
	if err == nil {
		t.Fatal("validateFeishuIdentitySourceConfig() error = nil, want blank verification token error")
	}
}

func TestValidateFeishuConfigAcceptsWebhookCredentialsWithoutOAuth(t *testing.T) {
	err := validateFeishuIdentitySourceConfig(false, []byte(`{"verification_token":"verify-token","encrypt_key":"encrypt-key"}`))
	if err != nil {
		t.Fatalf("validateFeishuIdentitySourceConfig() error = %v", err)
	}
}

func TestValidateFeishuConfigRejectsOversizedWebhookCredentials(t *testing.T) {
	value := string(bytes.Repeat([]byte("x"), 513))
	raw := []byte(`{"encrypt_key":"` + value + `"}`)
	if err := validateFeishuIdentitySourceConfig(false, raw); err == nil {
		t.Fatal("validateFeishuIdentitySourceConfig() error = nil, want oversized encrypt key error")
	}
}

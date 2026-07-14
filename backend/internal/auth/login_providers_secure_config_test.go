package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"

	"github.com/smices/open-idb/internal/db/generated"
	"github.com/smices/open-idb/internal/secureconfig"
)

func TestResolveFeishuConfigDecryptsStoredProviderConfig(t *testing.T) {
	codec, err := secureconfig.New(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x61}, 32)))
	if err != nil {
		t.Fatalf("secureconfig.New() error = %v", err)
	}
	sealed, err := codec.Seal([]byte(`{"app_id":"cli_encrypted","app_secret":"encrypted-secret"}`))
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	service := &LoginProviderService{
		queries: &mockProviderQueries{listFn: func(context.Context, string) ([]generated.ListLoginProvidersRow, error) {
			return []generated.ListLoginProvidersRow{{Provider: "feishu", Config: sealed}}, nil
		}},
		configCodec: codec,
	}

	got, err := service.ResolveFeishuConfig(context.Background(), "01HZZZZZZZ0000000000000001")
	if err != nil {
		t.Fatalf("ResolveFeishuConfig() error = %v", err)
	}
	if got.AppID != "cli_encrypted" || got.AppSecret != "encrypted-secret" {
		t.Fatalf("ResolveFeishuConfig() = %#v", got)
	}
}

// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestFeishuWebhookRejectsBodyAboveLimit(t *testing.T) {
	router := newTestRouter(&fakeSyncService{})
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu/entity-1/source-1", strings.NewReader(strings.Repeat("x", 1<<20+1)))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestFeishuWebhookRejectsInvalidVerificationToken(t *testing.T) {
	handler := NewHandler(&fakeSyncService{}, nil)
	handler.SetFeishuWebhookSecurityResolver(func(context.Context, string, string) (FeishuWebhookSecurityConfig, error) {
		return FeishuWebhookSecurityConfig{VerificationToken: "expected-token"}, nil
	})
	router := chi.NewRouter()
	handler.RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu/entity-1/source-1", strings.NewReader(`{
		"type":"url_verification",
		"challenge":"challenge-token",
		"token":"wrong-token"
	}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestFeishuWebhookLegacyCompatibilityWarnsWithoutBreakingChallenge(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	handler := NewHandler(&fakeSyncService{}, nil)
	handler.SetLogger(zap.New(core))
	handler.SetFeishuWebhookSecurityResolver(func(context.Context, string, string) (FeishuWebhookSecurityConfig, error) {
		return FeishuWebhookSecurityConfig{}, nil
	})
	router := chi.NewRouter()
	handler.RegisterRoutes(router)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu/entity-1/source-1", strings.NewReader(`{
		"type":"url_verification",
		"challenge":"legacy-challenge"
	}`))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if logs.FilterMessage("feishu webhook is running in legacy unverified compatibility mode").Len() != 1 {
		t.Fatalf("warning logs = %#v", logs.All())
	}
}

func TestFeishuWebhookAcceptsEncryptedChallengeWithoutSignature(t *testing.T) {
	const encryptKey = "test-encrypt-key"
	const verificationToken = "test-verification-token"
	handler := NewHandler(&fakeSyncService{}, nil)
	handler.SetFeishuWebhookSecurityResolver(func(context.Context, string, string) (FeishuWebhookSecurityConfig, error) {
		return FeishuWebhookSecurityConfig{VerificationToken: verificationToken, EncryptKey: encryptKey}, nil
	})
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	plain := []byte(`{"type":"url_verification","challenge":"encrypted-challenge","token":"test-verification-token"}`)
	body := encryptedFeishuWebhookBody(t, encryptKey, plain)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu/entity-1/source-1", strings.NewReader(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["challenge"] != "encrypted-challenge" {
		t.Fatalf("response = %#v", response)
	}
}

func TestFeishuWebhookRejectsUnsignedEncryptedEvent(t *testing.T) {
	const encryptKey = "test-encrypt-key"
	const verificationToken = "test-verification-token"
	handler := NewHandler(&fakeSyncService{}, nil)
	handler.SetFeishuWebhookSecurityResolver(func(context.Context, string, string) (FeishuWebhookSecurityConfig, error) {
		return FeishuWebhookSecurityConfig{VerificationToken: verificationToken, EncryptKey: encryptKey}, nil
	})
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	plain := []byte(`{"type":"event_callback","token":"test-verification-token","timestamp":"1710000000","event":{"event_type":"added_user","object_type":"user","object_id":"ou_123","event_id":"evt-unsigned"}}`)
	body := encryptedFeishuWebhookBody(t, encryptKey, plain)
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu/entity-1/source-1", strings.NewReader(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestFeishuWebhookAcceptsDelayedConfiguredEventForProviderRetry(t *testing.T) {
	service := &fakeSyncService{submitWebhookJobID: "webhook-job-delayed"}
	handler := NewHandler(service, nil)
	handler.SetFeishuWebhookSecurityResolver(func(context.Context, string, string) (FeishuWebhookSecurityConfig, error) {
		return FeishuWebhookSecurityConfig{VerificationToken: "test-token"}, nil
	})
	router := chi.NewRouter()
	handler.RegisterRoutes(router)
	body := fmt.Sprintf(`{
		"type":"event_callback",
		"token":"test-token",
		"timestamp":"%d",
		"event":{"event_type":"added_user","object_type":"user","object_id":"ou_123","event_id":"evt-stale"}
	}`, time.Now().Add(-10*time.Minute).Unix())
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu/entity-1/source-1", strings.NewReader(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if service.submitWebhookCalls != 1 {
		t.Fatalf("submit calls = %d, want 1", service.submitWebhookCalls)
	}
}

func TestFeishuWebhookAcceptsDelayedSignedEncryptedEvent(t *testing.T) {
	const encryptKey = "test-encrypt-key"
	const verificationToken = "test-verification-token"
	service := &fakeSyncService{submitWebhookJobID: "webhook-job-delayed-encrypted"}
	handler := NewHandler(service, nil)
	handler.SetFeishuWebhookSecurityResolver(func(context.Context, string, string) (FeishuWebhookSecurityConfig, error) {
		return FeishuWebhookSecurityConfig{VerificationToken: verificationToken, EncryptKey: encryptKey}, nil
	})
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	delayedTimestamp := fmt.Sprint(time.Now().Add(-6 * time.Hour).Unix())
	plain := []byte(fmt.Sprintf(`{"type":"event_callback","token":"test-verification-token","timestamp":%q,"event":{"event_type":"added_user","object_type":"user","object_id":"ou_delayed","event_id":"evt-delayed-encrypted"}}`, delayedTimestamp))
	body := encryptedFeishuWebhookBody(t, encryptKey, plain)
	nonce := "nonce-delayed"
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu/entity-1/source-1", strings.NewReader(body))
	req.Header.Set("X-Lark-Request-Timestamp", delayedTimestamp)
	req.Header.Set("X-Lark-Request-Nonce", nonce)
	req.Header.Set("X-Lark-Signature", feishuWebhookSignature(delayedTimestamp, nonce, encryptKey, []byte(body)))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if service.submitWebhookCalls != 1 {
		t.Fatalf("submit calls = %d, want 1", service.submitWebhookCalls)
	}
}

func TestFeishuWebhookDeduplicatesEventID(t *testing.T) {
	service := &fakeSyncService{submitWebhookJobID: "webhook-job-1"}
	router := newTestRouter(service)
	body := `{
		"type":"event_callback",
		"event":{"event_type":"added_user","object_type":"user","object_id":"ou_123","event_id":"evt-duplicate"}
	}`

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu/entity-1/source-1", strings.NewReader(body)))
	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu/entity-1/source-1", strings.NewReader(body)))

	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusOK)
	}
	if second.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d; body=%s", second.Code, http.StatusOK, second.Body.String())
	}
	if service.submitWebhookCalls != 1 {
		t.Fatalf("submit calls = %d, want 1", service.submitWebhookCalls)
	}
}

func TestFeishuWebhookLegacyModeIgnoresDeleteEvent(t *testing.T) {
	service := &fakeSyncService{submitWebhookJobID: "must-not-be-created"}
	router := newTestRouter(service)
	body := `{
		"type":"event_callback",
		"event":{"event_type":"delete_user","object_type":"user","object_id":"union_sensitive","object_id_type":"union_id","event_id":"evt-unverified-delete"}
	}`
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu/entity-1/source-1", strings.NewReader(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if service.submitWebhookCalls != 0 {
		t.Fatalf("submit calls = %d, want 0", service.submitWebhookCalls)
	}
}

func TestFeishuWebhookVerifiedModeIgnoresUntypedDeleteEvent(t *testing.T) {
	service := &fakeSyncService{submitWebhookJobID: "must-not-be-created"}
	handler := NewHandler(service, nil)
	handler.SetFeishuWebhookSecurityResolver(func(context.Context, string, string) (FeishuWebhookSecurityConfig, error) {
		return FeishuWebhookSecurityConfig{VerificationToken: "test-token"}, nil
	})
	router := chi.NewRouter()
	handler.RegisterRoutes(router)
	body := `{
		"type":"event_callback",
		"token":"test-token",
		"event":{"event_type":"delete_user","object_type":"user","object_id":"union_sensitive","event_id":"evt-untyped-delete"}
	}`
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu/entity-1/source-1", strings.NewReader(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if service.submitWebhookCalls != 0 {
		t.Fatalf("submit calls = %d, want 0", service.submitWebhookCalls)
	}
}

func TestFeishuWebhookVerifiedModeAcceptsTypedDeleteEvent(t *testing.T) {
	service := &fakeSyncService{submitWebhookJobID: "typed-delete-job"}
	handler := NewHandler(service, nil)
	handler.SetFeishuWebhookSecurityResolver(func(context.Context, string, string) (FeishuWebhookSecurityConfig, error) {
		return FeishuWebhookSecurityConfig{VerificationToken: "test-token"}, nil
	})
	router := chi.NewRouter()
	handler.RegisterRoutes(router)
	body := `{
		"type":"event_callback",
		"token":"test-token",
		"event":{"event_type":"delete_user","object_type":"user","object_id":"union_sensitive","object_id_type":"union_id","event_id":"evt-typed-delete"}
	}`
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/webhooks/feishu/entity-1/source-1", strings.NewReader(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if service.submitWebhookCalls != 1 {
		t.Fatalf("submit calls = %d, want 1", service.submitWebhookCalls)
	}
}

func TestFeishuWebhookRunnerIsBounded(t *testing.T) {
	runner := newFeishuWebhookRunner(1, time.Minute)
	if !runner.tryReserve() {
		t.Fatal("first reservation rejected")
	}
	if runner.tryReserve() {
		t.Fatal("second reservation accepted while capacity is full")
	}
	runner.release()
	if !runner.tryReserve() {
		t.Fatal("reservation rejected after release")
	}
	runner.release()
}

func TestParseFeishuWebhookV2UsesHeaderIdentity(t *testing.T) {
	event, err := parseFeishuWebhookEvent([]byte(`{
		"schema":"2.0",
		"header":{
			"event_id":"evt-v2",
			"event_type":"contact.user.updated_v3",
			"create_time":"1710000000000"
		},
		"event":{"object":{"user_id":"ou_v2"}}
	}`))
	if err != nil {
		t.Fatalf("parseFeishuWebhookEvent() error = %v", err)
	}
	if event.EventID != "evt-v2" || event.EventType != "contact.user.updated_v3" {
		t.Fatalf("event header fields = %#v", event)
	}
	if event.ObjectType != "user" || event.ObjectID != "ou_v2" || event.ObjectIDType != "user_id" {
		t.Fatalf("event object = %#v", event)
	}
}

func encryptedFeishuWebhookBody(t *testing.T, encryptKey string, plain []byte) string {
	t.Helper()
	key := sha256.Sum256([]byte(encryptKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		t.Fatalf("aes.NewCipher: %v", err)
	}
	padding := aes.BlockSize - len(plain)%aes.BlockSize
	padded := append(append([]byte(nil), plain...), bytesOf(byte(padding), padding)...)
	iv := []byte("0123456789abcdef")
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)
	encrypted := base64.StdEncoding.EncodeToString(append(append([]byte(nil), iv...), ciphertext...))
	body, err := json.Marshal(map[string]string{"encrypt": encrypted})
	if err != nil {
		t.Fatalf("marshal encrypted body: %v", err)
	}
	return string(body)
}

func feishuWebhookSignature(timestamp string, nonce string, encryptKey string, body []byte) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte(timestamp))
	_, _ = digest.Write([]byte(nonce))
	_, _ = digest.Write([]byte(encryptKey))
	_, _ = digest.Write(body)
	return hex.EncodeToString(digest.Sum(nil))
}

func bytesOf(value byte, count int) []byte {
	out := make([]byte, count)
	for i := range out {
		out[i] = value
	}
	return out
}

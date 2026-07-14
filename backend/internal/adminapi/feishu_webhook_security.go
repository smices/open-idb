// SPDX-License-Identifier: MIT

package adminapi

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	maxFeishuWebhookBodyBytes = int64(1 << 20)
	feishuWebhookDedupeTTL    = 24 * time.Hour
	feishuWebhookDedupeLimit  = 10_000
	// Sync one webhook batch at a time. The sync service itself serializes by
	// source; accepting concurrent runs here would acknowledge jobs that then
	// fail with "sync already running" instead of letting Feishu retry them.
	feishuWebhookConcurrency  = 1
	feishuWebhookSyncTimeout  = 2 * time.Minute
	legacyWebhookWarnInterval = time.Hour
)

// FeishuWebhookSecurityConfig contains the per-identity-source verification
// material configured in the Feishu developer console.
type FeishuWebhookSecurityConfig struct {
	VerificationToken string
	EncryptKey        string
}

// FeishuWebhookSecurityResolver resolves security material for the exact
// entity/source pair addressed by the webhook URL.
type FeishuWebhookSecurityResolver func(ctx context.Context, entityID, sourceID string) (FeishuWebhookSecurityConfig, error)

type feishuWebhookRuntime struct {
	resolver FeishuWebhookSecurityResolver
	logger   *zap.Logger
	now      func() time.Time
	runner   *feishuWebhookRunner
	deduper  *feishuWebhookDeduper

	warnMu     sync.Mutex
	lastWarnAt time.Time
}

func newFeishuWebhookRuntime() *feishuWebhookRuntime {
	return &feishuWebhookRuntime{
		now:     time.Now,
		runner:  newFeishuWebhookRunner(feishuWebhookConcurrency, feishuWebhookSyncTimeout),
		deduper: newFeishuWebhookDeduper(feishuWebhookDedupeLimit, feishuWebhookDedupeTTL),
	}
}

// SetFeishuWebhookSecurityResolver enables strict webhook verification when
// verification_token or encrypt_key is configured. A nil resolver keeps the
// legacy compatibility mode for existing deployments.
func (h *Handler) SetFeishuWebhookSecurityResolver(resolver FeishuWebhookSecurityResolver) {
	h.ensureFeishuWebhookRuntime().resolver = resolver
}

// SetLogger enables rate-limited compatibility and processing warnings.
func (h *Handler) SetLogger(logger *zap.Logger) {
	h.ensureFeishuWebhookRuntime().logger = logger
}

func (h *Handler) ensureFeishuWebhookRuntime() *feishuWebhookRuntime {
	if h.webhooks == nil {
		h.webhooks = newFeishuWebhookRuntime()
	}
	return h.webhooks
}

func (r *feishuWebhookRuntime) securityConfig(ctx context.Context, entityID, sourceID string) (FeishuWebhookSecurityConfig, error) {
	if r.resolver == nil {
		r.warnLegacyMode(entityID, sourceID, "security resolver is not configured")
		return FeishuWebhookSecurityConfig{}, nil
	}
	cfg, err := r.resolver(ctx, entityID, sourceID)
	if err != nil {
		return FeishuWebhookSecurityConfig{}, err
	}
	cfg.VerificationToken = strings.TrimSpace(cfg.VerificationToken)
	cfg.EncryptKey = strings.TrimSpace(cfg.EncryptKey)
	if cfg.VerificationToken == "" && cfg.EncryptKey == "" {
		r.warnLegacyMode(entityID, sourceID, "verification_token and encrypt_key are not configured")
	}
	return cfg, nil
}

func (r *feishuWebhookRuntime) warnLegacyMode(entityID, sourceID, reason string) {
	if r.logger == nil {
		return
	}
	now := r.now()
	r.warnMu.Lock()
	defer r.warnMu.Unlock()
	if !r.lastWarnAt.IsZero() && now.Sub(r.lastWarnAt) < legacyWebhookWarnInterval {
		return
	}
	r.lastWarnAt = now
	r.logger.Warn("feishu webhook is running in legacy unverified compatibility mode",
		zap.String("entity_id", entityID),
		zap.String("source_id", sourceID),
		zap.String("reason", reason),
	)
}

type webhookEnvelopeMetadata struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge"`
	Token     string `json:"token"`
	Encrypt   string `json:"encrypt"`
	Header    struct {
		Token string `json:"token"`
	} `json:"header"`
}

func prepareFeishuWebhookPayload(r *http.Request, raw []byte, cfg FeishuWebhookSecurityConfig) ([]byte, error) {
	payload := raw
	if cfg.EncryptKey != "" {
		decrypted, err := decryptFeishuWebhookPayload(raw, cfg.EncryptKey)
		if err != nil {
			return nil, err
		}
		payload = decrypted
	} else {
		var outer webhookEnvelopeMetadata
		if err := json.Unmarshal(raw, &outer); err != nil {
			return nil, newWebhookSecurityError(http.StatusBadRequest, "invalid_webhook_body", "invalid json body")
		}
		if strings.TrimSpace(outer.Encrypt) != "" {
			return nil, newWebhookSecurityError(http.StatusUnauthorized, "webhook_encryption_not_configured", "encrypted webhook cannot be verified")
		}
	}

	var metadata webhookEnvelopeMetadata
	if err := json.Unmarshal(payload, &metadata); err != nil {
		return nil, newWebhookSecurityError(http.StatusBadRequest, "invalid_webhook_body", "invalid json body")
	}
	// Feishu's encrypted URL verification challenge does not carry the normal
	// event-signature headers. Its verification token is checked below. All
	// other encrypted events must still pass the signature over the raw body.
	if cfg.EncryptKey != "" && !strings.EqualFold(metadata.Type, "url_verification") {
		if err := verifyFeishuWebhookSignature(r, raw, cfg.EncryptKey); err != nil {
			return nil, err
		}
	}
	if cfg.VerificationToken != "" {
		received := strings.TrimSpace(firstNonEmptyString(metadata.Token, metadata.Header.Token))
		if received == "" || subtle.ConstantTimeCompare([]byte(received), []byte(cfg.VerificationToken)) != 1 {
			return nil, newWebhookSecurityError(http.StatusUnauthorized, "invalid_webhook_token", "webhook verification token is invalid")
		}
	}
	return payload, nil
}

func verifyFeishuWebhookSignature(r *http.Request, body []byte, encryptKey string) error {
	timestamp := strings.TrimSpace(r.Header.Get("X-Lark-Request-Timestamp"))
	nonce := strings.TrimSpace(r.Header.Get("X-Lark-Request-Nonce"))
	signature := strings.TrimSpace(r.Header.Get("X-Lark-Signature"))
	if timestamp == "" || nonce == "" || signature == "" {
		return newWebhookSecurityError(http.StatusUnauthorized, "missing_webhook_signature", "webhook signature headers are required")
	}
	digest := sha256.New()
	_, _ = digest.Write([]byte(timestamp))
	_, _ = digest.Write([]byte(nonce))
	_, _ = digest.Write([]byte(encryptKey))
	_, _ = digest.Write(body)
	expected := hex.EncodeToString(digest.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(strings.ToLower(signature)), []byte(expected)) != 1 {
		return newWebhookSecurityError(http.StatusUnauthorized, "invalid_webhook_signature", "webhook signature is invalid")
	}
	return nil
}

func decryptFeishuWebhookPayload(raw []byte, encryptKey string) ([]byte, error) {
	var envelope struct {
		Encrypt string `json:"encrypt"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || strings.TrimSpace(envelope.Encrypt) == "" {
		return nil, newWebhookSecurityError(http.StatusBadRequest, "invalid_encrypted_webhook", "encrypted webhook body is invalid")
	}
	decoded, err := base64.StdEncoding.DecodeString(envelope.Encrypt)
	if err != nil || len(decoded) < 2*aes.BlockSize || (len(decoded)-aes.BlockSize)%aes.BlockSize != 0 {
		return nil, newWebhookSecurityError(http.StatusBadRequest, "invalid_encrypted_webhook", "encrypted webhook payload is invalid")
	}
	key := sha256.Sum256([]byte(encryptKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, newWebhookSecurityError(http.StatusBadRequest, "invalid_encrypted_webhook", "encrypted webhook payload is invalid")
	}
	iv := decoded[:aes.BlockSize]
	plaintext := append([]byte(nil), decoded[aes.BlockSize:]...)
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, plaintext)
	padding := int(plaintext[len(plaintext)-1])
	if padding == 0 || padding > aes.BlockSize || padding > len(plaintext) {
		return nil, newWebhookSecurityError(http.StatusBadRequest, "invalid_encrypted_webhook", "encrypted webhook padding is invalid")
	}
	for _, value := range plaintext[len(plaintext)-padding:] {
		if int(value) != padding {
			return nil, newWebhookSecurityError(http.StatusBadRequest, "invalid_encrypted_webhook", "encrypted webhook padding is invalid")
		}
	}
	plaintext = plaintext[:len(plaintext)-padding]
	if !json.Valid(plaintext) {
		return nil, newWebhookSecurityError(http.StatusBadRequest, "invalid_encrypted_webhook", "decrypted webhook payload is invalid")
	}
	return plaintext, nil
}

type webhookSecurityError struct {
	status  int
	code    string
	message string
}

func newWebhookSecurityError(status int, code, message string) error {
	return &webhookSecurityError{status: status, code: code, message: message}
}

func (e *webhookSecurityError) Error() string {
	return fmt.Sprintf("%s: %s", e.code, e.message)
}

type feishuWebhookRunner struct {
	slots   chan struct{}
	timeout time.Duration
}

func newFeishuWebhookRunner(concurrency int, timeout time.Duration) *feishuWebhookRunner {
	if concurrency < 1 {
		concurrency = 1
	}
	return &feishuWebhookRunner{slots: make(chan struct{}, concurrency), timeout: timeout}
}

func (r *feishuWebhookRunner) tryReserve() bool {
	select {
	case r.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (r *feishuWebhookRunner) release() {
	<-r.slots
}

func (r *feishuWebhookRunner) runReserved(fn func(context.Context)) {
	go func() {
		defer r.release()
		ctx := context.Background()
		if r.timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, r.timeout)
			defer cancel()
		}
		fn(ctx)
	}()
}

type feishuWebhookDeduper struct {
	mu          sync.Mutex
	entries     map[string]time.Time
	limit       int
	ttl         time.Duration
	nextCleanup time.Time
}

func newFeishuWebhookDeduper(limit int, ttl time.Duration) *feishuWebhookDeduper {
	return &feishuWebhookDeduper{entries: make(map[string]time.Time), limit: limit, ttl: ttl}
}

func (d *feishuWebhookDeduper) reserve(key string, now time.Time) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.nextCleanup.IsZero() || !now.Before(d.nextCleanup) {
		for existing, expiresAt := range d.entries {
			if !expiresAt.After(now) {
				delete(d.entries, existing)
			}
		}
		d.nextCleanup = now.Add(time.Minute)
	}
	if expiresAt, ok := d.entries[key]; ok && expiresAt.After(now) {
		return false
	}
	if d.limit > 0 && len(d.entries) >= d.limit {
		// All reservations share the same TTL and expired entries were already
		// cleaned above. Evict one arbitrary entry in O(1) average time instead
		// of scanning the entire bounded map for every new event at capacity.
		for existing := range d.entries {
			delete(d.entries, existing)
			break
		}
	}
	d.entries[key] = now.Add(d.ttl)
	return true
}

func (d *feishuWebhookDeduper) remove(key string) {
	d.mu.Lock()
	delete(d.entries, key)
	d.mu.Unlock()
}

func feishuWebhookDedupeKey(entityID, sourceID string, eventID string, payload []byte) string {
	identity := strings.TrimSpace(eventID)
	if identity == "" {
		digest := sha256.Sum256(payload)
		identity = hex.EncodeToString(digest[:])
	}
	return entityID + ":" + sourceID + ":" + identity
}

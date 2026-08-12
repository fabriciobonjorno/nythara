package mailer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"testing"
	"time"
)

func TestResendWebhookVerifierUsesRawBodyAndRejectsReplayWindow(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	verifier, err := NewResendWebhookVerifier("whsec_" + base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	timestamp := strconv.FormatInt(now.Unix(), 10)
	id := "msg_webhook_123"
	raw := []byte(`{"type":"email.delivered"}`)
	signature := signWebhookTest(key, id, timestamp, raw)

	if err := verifier.Verify(raw, id, timestamp, "v1,not-base64 v1,"+signature, now); err != nil {
		t.Fatalf("assinatura válida: %v", err)
	}
	if err := verifier.Verify([]byte(`{"type": "email.delivered"}`), id, timestamp, "v1,"+signature, now); err == nil {
		t.Fatal("corpo reserializado deveria invalidar a assinatura")
	}
	if err := verifier.Verify(raw, id, timestamp, "v1,"+signature, now.Add(6*time.Minute)); err == nil {
		t.Fatal("assinatura fora da janela deveria ser rejeitada")
	}
}

func TestResendWebhookVerifierRejectsMalformedSecret(t *testing.T) {
	for _, secret := range []string{"", "secret", "whsec_invalido"} {
		if _, err := NewResendWebhookVerifier(secret); err == nil {
			t.Fatalf("segredo %q deveria falhar", secret)
		}
	}
}

func signWebhookTest(key []byte, id, timestamp string, raw []byte) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(id + "." + timestamp + "." + string(raw)))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

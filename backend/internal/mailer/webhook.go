package mailer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
)

const webhookTimestampTolerance = 5 * time.Minute

var ErrInvalidWebhook = errors.New("webhook do Resend inválido")

// ResendWebhookVerifier implementa o envelope assinado Svix usado pelo
// Resend. A assinatura sempre cobre o corpo bruto, sem reserializar o JSON.
type ResendWebhookVerifier struct {
	key       []byte
	tolerance time.Duration
}

func NewResendWebhookVerifier(secret string) (*ResendWebhookVerifier, error) {
	secret = strings.TrimSpace(secret)
	if !strings.HasPrefix(secret, "whsec_") {
		return nil, errors.New("RESEND_WEBHOOK_SECRET deve começar com whsec_")
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	if err != nil {
		key, err = base64.RawStdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	}
	if err != nil || len(key) < 16 {
		return nil, errors.New("RESEND_WEBHOOK_SECRET inválido")
	}
	return &ResendWebhookVerifier{key: key, tolerance: webhookTimestampTolerance}, nil
}

func (v *ResendWebhookVerifier) Verify(raw []byte, messageID, timestamp, signatures string, now time.Time) error {
	if v == nil || messageID == "" || len(messageID) > 200 || timestamp == "" || signatures == "" {
		return ErrInvalidWebhook
	}
	unixSeconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return ErrInvalidWebhook
	}
	delta := now.UTC().Sub(time.Unix(unixSeconds, 0).UTC())
	if delta < -v.tolerance || delta > v.tolerance {
		return ErrInvalidWebhook
	}
	signed := messageID + "." + timestamp + "." + string(raw)
	mac := hmac.New(sha256.New, v.key)
	_, _ = mac.Write([]byte(signed))
	expected := mac.Sum(nil)
	for _, candidate := range strings.Fields(signatures) {
		version, encoded, ok := strings.Cut(candidate, ",")
		if !ok || version != "v1" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err == nil && hmac.Equal(decoded, expected) {
			return nil
		}
	}
	return ErrInvalidWebhook
}

package mailer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func TestResendSendsPasswordResetWithoutLeakingKey(t *testing.T) {
	var authorization string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		authorization = r.Header.Get("Authorization")
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["from"] != "Nythara <no-reply@nythara.fun>" || !strings.Contains(payload["html"].(string), "https://nythara.fun/reset-password?token=segredo") {
			t.Fatalf("payload inesperado: %+v", payload)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"email-id"}`))}, nil
	})}

	sender, err := newResend("re_test_secret", "Nythara <no-reply@nythara.fun>", "https://api.resend.test/emails", client)
	if err != nil {
		t.Fatal(err)
	}
	if err := sender.SendPasswordReset(context.Background(), "player@example.test",
		"https://nythara.fun/reset-password?token=segredo", "pt-BR", 30*time.Minute); err != nil {
		t.Fatal(err)
	}
	if authorization != "Bearer re_test_secret" {
		t.Fatalf("authorization=%q", authorization)
	}
}

func TestResendReportsProviderFailure(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"message":"invalid api key"}`))}, nil
	})}
	sender, _ := newResend("secret", "Nythara <no-reply@nythara.fun>", "https://api.resend.test/emails", client)
	err := sender.SendPasswordReset(context.Background(), "player@example.test", "https://nythara.fun/reset", "en", 30*time.Minute)
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("erro inseguro ou ausente: %v", err)
	}
}

package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"
)

const resendEndpoint = "https://api.resend.com/emails"

// Resend envia somente mensagens transacionais de autenticação. A chave fica
// no ambiente da API e jamais é devolvida ao cliente ou incluída nos logs.
type Resend struct {
	apiKey   string
	from     string
	endpoint string
	client   *http.Client
}

func NewResend(apiKey, from string) (*Resend, error) {
	return newResend(apiKey, from, resendEndpoint, &http.Client{Timeout: 10 * time.Second})
}

func newResend(apiKey, from, endpoint string, client *http.Client) (*Resend, error) {
	apiKey, from = strings.TrimSpace(apiKey), strings.TrimSpace(from)
	if apiKey == "" || from == "" {
		return nil, errors.New("RESEND_API_KEY e RESEND_FROM_EMAIL são obrigatórios")
	}
	if client == nil {
		return nil, errors.New("cliente HTTP do Resend ausente")
	}
	return &Resend{apiKey: apiKey, from: from, endpoint: endpoint, client: client}, nil
}

func (r *Resend) SendPasswordReset(ctx context.Context, to, link, locale string, ttl time.Duration) error {
	subject, heading, intro, action, warning := resetCopy(locale, ttl)
	escapedLink := html.EscapeString(link)
	htmlBody := fmt.Sprintf(`<!doctype html><html><body style="margin:0;background:#0b080b;color:#f2e8d8;font-family:Arial,sans-serif"><div style="max-width:560px;margin:0 auto;padding:40px 24px"><p style="color:#c6a75b;letter-spacing:.18em;font-size:12px">NYTHARA</p><h1 style="font-family:Georgia,serif;font-weight:400">%s</h1><p style="line-height:1.65;color:#cfc3b9">%s</p><p style="margin:32px 0"><a href="%s" style="display:inline-block;background:#a93650;color:#fff8ef;padding:14px 22px;text-decoration:none;font-weight:700">%s</a></p><p style="font-size:13px;line-height:1.6;color:#8f8388">%s</p><p style="font-size:12px;word-break:break-all;color:#6f6569">%s</p></div></body></html>`,
		html.EscapeString(heading), html.EscapeString(intro), escapedLink,
		html.EscapeString(action), html.EscapeString(warning), escapedLink)
	textBody := fmt.Sprintf("NYTHARA\n\n%s\n\n%s\n\n%s\n\n%s", heading, intro, link, warning)
	payload, err := json.Marshal(map[string]any{
		"from": r.from, "to": []string{to}, "subject": subject,
		"html": htmlBody, "text": textBody,
	})
	if err != nil {
		return fmt.Errorf("codificar e-mail de recuperação: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("criar requisição Resend: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")
	response, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("enviar e-mail pelo Resend: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		// O corpo de erro do provedor pode repetir dados da mensagem. O status é
		// suficiente para operação sem levar endereço ou conteúdo aos logs.
		return fmt.Errorf("Resend respondeu status %d", response.StatusCode)
	}
	return nil
}

func resetCopy(locale string, ttl time.Duration) (subject, heading, intro, action, warning string) {
	minutes := int(ttl / time.Minute)
	switch locale {
	case "es":
		return "Restablece tu contraseña de Nythara", "Restablece tu contraseña",
			"Recibimos una solicitud para cambiar tu contraseña de Nythara.", "Elegir nueva contraseña",
			fmt.Sprintf("Este enlace caduca en %d minutos y solo se puede usar una vez. Si no hiciste la solicitud, ignora este mensaje.", minutes)
	case "en":
		return "Reset your Nythara password", "Reset your password",
			"We received a request to change your Nythara password.", "Choose a new password",
			fmt.Sprintf("This link expires in %d minutes and can only be used once. If you did not request it, ignore this message.", minutes)
	default:
		return "Redefina sua senha do Nythara", "Redefina sua senha",
			"Recebemos um pedido para trocar a senha da sua conta Nythara.", "Escolher nova senha",
			fmt.Sprintf("Este link expira em %d minutos e só pode ser usado uma vez. Se você não fez o pedido, ignore esta mensagem.", minutes)
	}
}

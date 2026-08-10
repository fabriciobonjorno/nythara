package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	handler, _ := testHandler()
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest("GET", "/healthz", nil))
	want := map[string]string{
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "no-referrer",
	}
	for header, value := range want {
		if got := resp.Header().Get(header); got != value {
			t.Fatalf("%s = %q; esperado %q", header, got, value)
		}
	}
}

func TestAuthEndpointsAreStrictlyLimited(t *testing.T) {
	handler, _ := testHandler()
	status := func() int {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/v1/auth/login", strings.NewReader(`{}`))
		req.RemoteAddr = "203.0.113.7:9999"
		handler.ServeHTTP(resp, req)
		return resp.Code
	}
	limited := false
	for i := 0; i < authBurst+2; i++ {
		if status() == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("força bruta de login deveria ser limitada dentro da rajada")
	}
	// Outro IP não é afetado.
	resp := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/auth/login", strings.NewReader(`{}`))
	req.RemoteAddr = "203.0.113.8:9999"
	handler.ServeHTTP(resp, req)
	if resp.Code == http.StatusTooManyRequests {
		t.Fatal("limite deve ser por IP")
	}
}

func TestMetricsEndpointServesCounters(t *testing.T) {
	handler, _ := testHandler()
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest("GET", "/internal/metrics", nil))
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "http_requests_total") {
		t.Fatalf("métricas indisponíveis: %d %s", resp.Code, resp.Body.String()[:120])
	}
}

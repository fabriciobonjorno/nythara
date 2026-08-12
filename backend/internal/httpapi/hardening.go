package httpapi

import (
	"expvar"
	"fmt"
	"net"
	"net/http"
	"strings"

	"veurubro/backend/internal/security"
)

// Fase 8 — endurecimento do perímetro HTTP/WS: cabeçalhos de segurança,
// rate limiting por IP (com faixa estrita para autenticação) e contadores
// de observabilidade expostos em /internal/metrics.

var (
	metricRequests   = expvar.NewInt("http_requests_total")
	metricLimited    = expvar.NewInt("http_rate_limited_total")
	metricAuthDenied = expvar.NewInt("http_auth_denied_total")
	metricPanics     = expvar.NewInt("http_panics_total")
	metricWSFlood    = expvar.NewInt("ws_command_flood_closed_total")
)

const (
	generalPerMinute = 240 // por IP, todas as rotas
	generalBurst     = 80
	authPerMinute    = 10 // por IP: login/register/refresh (força bruta)
	authBurst        = 5
	wsCmdPerMinute   = 120 // por conexão de batalha (command spam)
	wsCmdBurst       = 30
)

// harden aplica cabeçalhos de segurança e o limite geral por IP. A API é
// JSON-pura: CSP default-src 'none' bloqueia qualquer interpretação ativa.
func (a *API) harden(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if r.TLS != nil {
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		metricRequests.Add(1)
		if !a.generalLimiter.Allow(clientIP(r, a.trustedProxies)) {
			metricLimited.Add(1)
			w.Header().Set("Retry-After", "30")
			writeError(w, http.StatusTooManyRequests, "rate_limited", "muitas requisições; aguarde")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// strictLimit protege endpoints de autenticação contra força bruta.
func (a *API) strictLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.authLimiter.Allow(clientIP(r, a.trustedProxies)) {
			metricLimited.Add(1)
			w.Header().Set("Retry-After", "60")
			writeError(w, http.StatusTooManyRequests, "rate_limited",
				"muitas tentativas de autenticação; aguarde")
			return
		}
		next(w, r)
	}
}

// clientIP só considera X-Forwarded-For quando a conexão veio diretamente de
// um proxy explicitamente confiável. A busca da direita para a esquerda evita
// que um cliente injete o primeiro valor do cabeçalho para escapar do limite.
func clientIP(r *http.Request, trustedProxies []*net.IPNet) string {
	peer := hostOnly(r.RemoteAddr)
	peerIP := net.ParseIP(peer)
	if peerIP == nil || !ipInNetworks(peerIP, trustedProxies) {
		return peer
	}
	forwarded := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for index := len(forwarded) - 1; index >= 0; index-- {
		candidate := strings.TrimSpace(forwarded[index])
		ip := net.ParseIP(candidate)
		if ip != nil && !ipInNetworks(ip, trustedProxies) {
			return candidate
		}
	}
	return peer
}

func hostOnly(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err == nil {
		return host
	}
	return address
}

func ipInNetworks(ip net.IP, networks []*net.IPNet) bool {
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func parseTrustedProxyCIDRs(raw string) ([]*net.IPNet, error) {
	fields := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' || r == '\t' })
	networks := make([]*net.IPNet, 0, len(fields))
	for _, field := range fields {
		_, network, err := net.ParseCIDR(field)
		if err != nil {
			return nil, fmt.Errorf("TRUSTED_PROXY_CIDRS contém %q inválido: %w", field, err)
		}
		networks = append(networks, network)
	}
	return networks, nil
}

func newGeneralLimiter() *security.RateLimiter {
	return security.NewRateLimiter(generalPerMinute, generalBurst)
}

func newAuthLimiter() *security.RateLimiter {
	return security.NewRateLimiter(authPerMinute, authBurst)
}

func newWSCommandLimiter() *security.RateLimiter {
	return security.NewRateLimiter(wsCmdPerMinute, wsCmdBurst)
}

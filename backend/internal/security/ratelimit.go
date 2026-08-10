package security

import (
	"sync"
	"time"
)

// RateLimiter é um token bucket por chave (IP, usuário ou conexão), com poda
// de chaves ociosas embutida. Thread-safe; o relógio é injetável para testes.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens por segundo
	burst   float64
	now     func() time.Time

	lastPrune time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter cria um limitador de perMinute requisições por minuto com
// tolerância a rajadas de burst.
func NewRateLimiter(perMinute, burst int) *RateLimiter {
	if perMinute < 1 {
		perMinute = 1
	}
	if burst < 1 {
		burst = 1
	}
	return &RateLimiter{
		buckets: map[string]*bucket{},
		rate:    float64(perMinute) / 60.0,
		burst:   float64(burst),
		now:     time.Now,
	}
}

// WithClock troca o relógio (testes).
func (l *RateLimiter) WithClock(now func() time.Time) *RateLimiter {
	l.now = now
	return l
}

// Allow consome um token da chave; false = limitada agora.
func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.maybePrune(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens = min(l.burst, b.tokens+elapsed*l.rate)
		b.last = now
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// maybePrune remove chaves cheias e ociosas para conter memória sob abuso de
// muitas origens distintas. Roda no máximo a cada minuto sob o lock corrente.
func (l *RateLimiter) maybePrune(now time.Time) {
	if now.Sub(l.lastPrune) < time.Minute && len(l.buckets) < 100_000 {
		return
	}
	l.lastPrune = now
	idle := 10 * time.Minute
	if len(l.buckets) >= 100_000 {
		idle = time.Minute
	}
	for key, b := range l.buckets {
		if now.Sub(b.last) > idle {
			delete(l.buckets, key)
		}
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

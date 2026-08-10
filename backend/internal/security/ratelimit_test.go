package security

import (
	"fmt"
	"testing"
	"time"
)

func TestRateLimiterBurstAndRefill(t *testing.T) {
	clock := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	limiter := NewRateLimiter(60, 3).WithClock(func() time.Time { return clock })

	for i := 0; i < 3; i++ {
		if !limiter.Allow("ip1") {
			t.Fatalf("rajada %d deveria passar", i)
		}
	}
	if limiter.Allow("ip1") {
		t.Fatal("quarta na mesma janela deveria ser limitada")
	}
	// Outra chave não é afetada.
	if !limiter.Allow("ip2") {
		t.Fatal("chaves independentes")
	}
	// 1 segundo repõe 1 token (60/min).
	clock = clock.Add(time.Second)
	if !limiter.Allow("ip1") {
		t.Fatal("token deveria ter sido reposto")
	}
	if limiter.Allow("ip1") {
		t.Fatal("apenas 1 token reposto")
	}
	// Reposição nunca passa do burst.
	clock = clock.Add(time.Hour)
	for i := 0; i < 3; i++ {
		if !limiter.Allow("ip1") {
			t.Fatalf("burst cheio após ociosidade (%d)", i)
		}
	}
	if limiter.Allow("ip1") {
		t.Fatal("burst não deve exceder o teto")
	}
}

func TestRateLimiterPrunesIdleKeys(t *testing.T) {
	clock := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	limiter := NewRateLimiter(60, 2).WithClock(func() time.Time { return clock })
	for i := 0; i < 1000; i++ {
		limiter.Allow(fmt.Sprintf("ip-%d", i))
	}
	if len(limiter.buckets) != 1000 {
		t.Fatalf("buckets: %d", len(limiter.buckets))
	}
	clock = clock.Add(11 * time.Minute)
	limiter.Allow("fresh")
	if len(limiter.buckets) > 2 {
		t.Fatalf("poda deveria ter removido ociosas; restam %d", len(limiter.buckets))
	}
}

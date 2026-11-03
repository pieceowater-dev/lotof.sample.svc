package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "app/internal/core/grpc/generated/generic/utils"
	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// readyAfter says how many readiness checks report "not ready" before the
// tenant becomes ready; a negative value means never.
type stubTenantConfig struct {
	readyAfter int
	// migrationStatus is what GetTenantMigrationInfo reports; "" means this
	// service has no record of the namespace at all.
	migrationStatus string
	checks          int
	checkErr        error
	warmups         int
	ensures         int
	ensureErr       error
	readyOnEnsure   bool
}

func (s *stubTenantConfig) IsTenantReady(context.Context, gossiper.Database, string) (bool, error) {
	s.checks++
	if s.checkErr != nil {
		return false, s.checkErr
	}
	if s.readyAfter < 0 {
		return false, nil
	}
	return s.checks > s.readyAfter, nil
}

func (s *stubTenantConfig) GetTenantMigrationInfo(context.Context, gossiper.Database, string) (string, string, error) {
	return s.migrationStatus, "", nil
}

func (s *stubTenantConfig) TriggerTenantWarmupAsync() { s.warmups++ }

func (s *stubTenantConfig) EnsureAppTenant(context.Context, string, string) error {
	s.ensures++
	if s.ensureErr == nil && s.readyOnEnsure {
		s.readyAfter = 0
	}
	return s.ensureErr
}

func tenantCtx(namespace, userID string) context.Context {
	return context.WithValue(context.Background(), UserMetadataKey,
		&pb.UserMetadata{Namespace: namespace, UserId: userID})
}

func callReadiness(t *testing.T, cfg *stubTenantConfig, ctx context.Context, maxWait time.Duration) (bool, error) {
	t.Helper()
	handled := false
	handler := func(context.Context, any) (any, error) {
		handled = true
		return "ok", nil
	}
	icpt := newTenantReadinessMiddleware(cfg, nil, []string{"/exempt.Service/Method"}, maxWait).MiddlewareMethod()
	_, err := icpt(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/plans.Service/List"}, handler)
	return handled, err
}

func TestTenantReadiness_ReadyTenantPassesStraightThrough(t *testing.T) {
	cfg := &stubTenantConfig{readyAfter: 0}
	handled, err := callReadiness(t, cfg, tenantCtx("ns", "user-1"), time.Second)
	if err != nil || !handled {
		t.Fatalf("expected the handler to run, got handled=%v err=%v", handled, err)
	}
	if cfg.warmups != 0 || cfg.ensures != 0 {
		t.Fatalf("a ready tenant must not trigger warmup or provisioning, got %d/%d", cfg.warmups, cfg.ensures)
	}
}

func TestTenantReadiness_UnknownTenantGivesUpWithinBudget(t *testing.T) {
	cfg := &stubTenantConfig{readyAfter: -1, migrationStatus: "running"}
	start := time.Now()
	handled, err := callReadiness(t, cfg, tenantCtx("ns-nobody-has", ""), 300*time.Millisecond)
	elapsed := time.Since(start)

	if handled {
		t.Fatal("the handler must not run for a tenant that never becomes ready")
	}
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("expected Unavailable, got %v", err)
	}
	// The old loop ran until the caller's deadline (60s in prod).
	if elapsed > 2*time.Second {
		t.Fatalf("waited %s, which is far past the 300ms budget", elapsed)
	}
	if cfg.warmups != 1 {
		t.Fatalf("warmup must be triggered exactly once per request, got %d", cfg.warmups)
	}
}

func TestTenantReadiness_RetriesWhileMigrationFinishes(t *testing.T) {
	cfg := &stubTenantConfig{readyAfter: 2, migrationStatus: "running"}
	handled, err := callReadiness(t, cfg, tenantCtx("ns", ""), 2*time.Second)
	if err != nil || !handled {
		t.Fatalf("expected the handler to run after the tenant became ready, got handled=%v err=%v", handled, err)
	}
	if cfg.checks < 3 {
		t.Fatalf("expected the readiness check to be retried, got %d checks", cfg.checks)
	}
}

func TestTenantReadiness_FirstConnectionProvisionsThroughHub(t *testing.T) {
	cfg := &stubTenantConfig{readyAfter: -1, readyOnEnsure: true, migrationStatus: "running"}
	handled, err := callReadiness(t, cfg, tenantCtx("ns-new", "user-1"), time.Second)
	if err != nil || !handled {
		t.Fatalf("expected provisioning to unblock the request, got handled=%v err=%v", handled, err)
	}
	if cfg.ensures != 1 {
		t.Fatalf("expected exactly one provisioning attempt, got %d", cfg.ensures)
	}
}

func TestTenantReadiness_AnonymousCallerIsNotProvisioned(t *testing.T) {
	cfg := &stubTenantConfig{readyAfter: -1, readyOnEnsure: true, migrationStatus: "running"}
	if _, err := callReadiness(t, cfg, tenantCtx("ns-new", ""), 300*time.Millisecond); status.Code(err) != codes.Unavailable {
		t.Fatalf("expected Unavailable, got %v", err)
	}
	if cfg.ensures != 0 {
		t.Fatal("provisioning must be scoped to a caller's own membership, never done for an anonymous call")
	}
}

func TestTenantReadiness_CheckErrorIsReported(t *testing.T) {
	cfg := &stubTenantConfig{checkErr: errors.New("db down"), migrationStatus: "running"}
	_, err := callReadiness(t, cfg, tenantCtx("ns", ""), 200*time.Millisecond)
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("expected Unavailable, got %v", err)
	}
	if got := status.Convert(err).Message(); got == "" || !contains(got, "db down") {
		t.Fatalf("the underlying error should stay visible, got %q", got)
	}
}

func TestTenantReadiness_UnprovisionedNamespaceFailsFast(t *testing.T) {
	// No migration record: waiting cannot help, and FailedPrecondition is a
	// code the gateway's read retries leave alone.
	cfg := &stubTenantConfig{readyAfter: -1, migrationStatus: ""}
	start := time.Now()
	handled, err := callReadiness(t, cfg, tenantCtx("ns-made-up", ""), 3*time.Second)
	elapsed := time.Since(start)

	if handled {
		t.Fatal("the handler must not run for a namespace this service does not have")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", err)
	}
	if elapsed > time.Second {
		t.Fatalf("an unprovisioned namespace must not consume the wait budget, took %s", elapsed)
	}
	if cfg.warmups != 1 {
		t.Fatalf("warmup should still be triggered once, got %d", cfg.warmups)
	}
}

func TestTenantReadiness_ExemptMethodSkipsTheGate(t *testing.T) {
	cfg := &stubTenantConfig{readyAfter: -1}
	handler := func(context.Context, any) (any, error) { return "ok", nil }
	icpt := newTenantReadinessMiddleware(cfg, nil, []string{"/exempt.Service/Method"}, time.Second).MiddlewareMethod()
	if _, err := icpt(tenantCtx("ns", ""), nil, &grpc.UnaryServerInfo{FullMethod: "/exempt.Service/Method"}, handler); err != nil {
		t.Fatalf("an exempt method must not be gated, got %v", err)
	}
	if cfg.checks != 0 {
		t.Fatalf("an exempt method must not be checked, got %d checks", cfg.checks)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

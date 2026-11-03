package internal

import (
	"context"
	"os"
	"time"
)

// startTenantWarmupLoop periodically re-discovers namespaces registered for
// this app and re-provisions/re-migrates any that changed -- a namespace
// added to this app after startup, or a schema change deployed while a
// tenant's process was already up, still gets picked up without a restart.
func (a *App) startTenantWarmupLoop() {
	interval := 10 * time.Minute
	if raw := os.Getenv("TENANT_MIGRATION_POLL_INTERVAL"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			interval = parsed
		}
	}

	loopCtx, cancel := context.WithCancel(context.Background())
	a.warmupCancel = cancel

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				a.cfg.TriggerTenantWarmupAsync()
			case <-loopCtx.Done():
				return
			}
		}
	}()
}

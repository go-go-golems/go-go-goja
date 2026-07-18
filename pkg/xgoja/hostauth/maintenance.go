package hostauth

import (
	"context"
	"fmt"
	"time"
)

// MaintenanceOptions describes a scheduled retention invocation. The caller
// owns scheduling and operator authorization; this operation never enters JS.
type MaintenanceOptions struct{ Before time.Time }

// RunMaintenanceCommand is the host-side retention command entry point used by
// cron/jobs and embedding applications.
func RunMaintenanceCommand(ctx context.Context, services *Services, opts MaintenanceOptions) (int, error) {
	if services == nil {
		return 0, fmt.Errorf("hostauth services are required")
	}
	if opts.Before.IsZero() {
		opts.Before = time.Now().UTC().Add(-24 * time.Hour)
	}
	return services.Maintenance.PurgeExpired(ctx, opts.Before)
}

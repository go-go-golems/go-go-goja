package main

import (
	"context"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/hashiplugin/sdk"
)

func main() {
	sdk.Serve(
		sdk.MustModule(
			"plugin:clock",
			sdk.Version("v1"),
			sdk.Doc("Example plugin that returns structured time snapshots"),
			sdk.Capabilities("examples", "metadata", "structured-results"),
			sdk.Function("now", func(_ context.Context, _ *sdk.Call) (any, error) {
				return snapshot(time.Now()), nil
			}, sdk.ExportDoc("Return the current local time as a structured object")),
			sdk.Object("utc",
				sdk.ObjectDoc("UTC time helpers"),
				sdk.Method("now", func(_ context.Context, _ *sdk.Call) (any, error) {
					return snapshot(time.Now().UTC()), nil
				}, sdk.ExportDoc("Return the current UTC time as a structured object")),
			),
		),
	)
}

func snapshot(t time.Time) map[string]any {
	return map[string]any{
		"unix":       t.Unix(),
		"rfc3339":    t.Format(time.RFC3339),
		"date":       t.Format("2006-01-02"),
		"time":       t.Format("15:04:05"),
		"zone":       t.Format("MST"),
		"weekday":    t.Weekday().String(),
		"nanosecond": t.Nanosecond(),
	}
}

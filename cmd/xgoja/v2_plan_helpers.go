package main

import (
	"strings"

	"github.com/go-go-golems/go-go-goja/cmd/xgoja/internal/plan"
)

type v2PlanTarget struct {
	Kind     string
	Output   string
	Package  string
	Template string
}

func targetFromPlan(compiled *plan.Plan) v2PlanTarget {
	if compiled == nil {
		return v2PlanTarget{Kind: "xgoja", Output: "dist/xgoja-app"}
	}
	for _, artifact := range compiled.Artifacts {
		spec := artifact.Spec
		kind := strings.TrimSpace(spec.Type)
		if kind == "" || kind == "dts" || kind == "embedded-assets" {
			continue
		}
		if kind == "binary" {
			kind = "xgoja"
		}
		if kind == "runtime-package" {
			kind = "package"
		}
		return v2PlanTarget{Kind: kind, Output: spec.Output, Package: spec.Package, Template: spec.Template}
	}
	return v2PlanTarget{Kind: "xgoja", Output: "dist/xgoja-app"}
}

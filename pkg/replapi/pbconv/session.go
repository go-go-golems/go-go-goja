package pbconv

import (
	replapiv1 "github.com/go-go-golems/go-go-goja/pkg/replapi/pb/proto/goja/replapi/v1"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
)

func EvaluateRequestFromProto(in *replapiv1.EvaluateRequest) replsession.EvaluateRequest {
	if in == nil {
		return replsession.EvaluateRequest{}
	}
	return replsession.EvaluateRequest{Source: in.GetSource()}
}

func EvaluateResponseToProto(in *replsession.EvaluateResponse) *replapiv1.EvaluateResponse {
	if in == nil {
		return nil
	}
	return &replapiv1.EvaluateResponse{SchemaVersion: SchemaVersion, Session: SessionSummaryToProto(in.Session), Cell: CellReportToProto(in.Cell)}
}

func SessionSummaryToProto(in *replsession.SessionSummary) *replapiv1.SessionSummary {
	if in == nil {
		return nil
	}
	return &replapiv1.SessionSummary{
		Id:             in.ID,
		Profile:        in.Profile,
		Policy:         SessionPolicyToProto(in.Policy),
		CreatedAt:      timestamp(in.CreatedAt),
		CellCount:      uint32(in.CellCount),
		BindingCount:   uint32(in.BindingCount),
		Bindings:       BindingViewsToProto(in.Bindings),
		History:        HistoryEntriesToProto(in.History),
		CurrentGlobals: GlobalStateViewsToProto(in.CurrentGlobals),
		Provenance:     ProvenanceRecordsToProto(in.Provenance),
	}
}

func SessionPolicyToProto(in replsession.SessionPolicy) *replapiv1.SessionPolicy {
	return &replapiv1.SessionPolicy{Eval: EvalPolicyToProto(in.Eval), Observe: ObservePolicyToProto(in.Observe), Persist: PersistPolicyToProto(in.Persist)}
}

func EvalPolicyToProto(in replsession.EvalPolicy) *replapiv1.EvalPolicy {
	return &replapiv1.EvalPolicy{Mode: EvalModeToProto(in.Mode), CaptureLastExpression: in.CaptureLastExpression, SupportTopLevelAwait: in.SupportTopLevelAwait, TimeoutMs: in.TimeoutMS}
}

func EvalModeToProto(in replsession.EvalMode) replapiv1.EvalMode {
	switch in {
	case replsession.EvalModeRaw:
		return replapiv1.EvalMode_EVAL_MODE_RAW
	case replsession.EvalModeInstrumented:
		return replapiv1.EvalMode_EVAL_MODE_INSTRUMENTED
	default:
		return replapiv1.EvalMode_EVAL_MODE_UNSPECIFIED
	}
}

func EvalModeFromProto(in replapiv1.EvalMode) replsession.EvalMode {
	switch in {
	case replapiv1.EvalMode_EVAL_MODE_UNSPECIFIED:
		return ""
	case replapiv1.EvalMode_EVAL_MODE_RAW:
		return replsession.EvalModeRaw
	case replapiv1.EvalMode_EVAL_MODE_INSTRUMENTED:
		return replsession.EvalModeInstrumented
	default:
		return ""
	}
}

func ObservePolicyToProto(in replsession.ObservePolicy) *replapiv1.ObservePolicy {
	return &replapiv1.ObservePolicy{StaticAnalysis: in.StaticAnalysis, RuntimeSnapshot: in.RuntimeSnapshot, BindingTracking: in.BindingTracking, ConsoleCapture: in.ConsoleCapture, JsdocExtraction: in.JSDocExtraction}
}

func PersistPolicyToProto(in replsession.PersistPolicy) *replapiv1.PersistPolicy {
	return &replapiv1.PersistPolicy{Enabled: in.Enabled, Evaluations: in.Evaluations, BindingVersions: in.BindingVersions, BindingDocs: in.BindingDocs}
}

func CellReportToProto(in *replsession.CellReport) *replapiv1.CellReport {
	if in == nil {
		return nil
	}
	return &replapiv1.CellReport{
		Id:           uint32(in.ID),
		CreatedAt:    timestamp(in.CreatedAt),
		Source:       in.Source,
		StaticReport: StaticReportToProto(in.Static),
		Rewrite:      RewriteReportToProto(in.Rewrite),
		Execution:    ExecutionReportToProto(in.Execution),
		Runtime:      RuntimeReportToProto(in.Runtime),
		Provenance:   ProvenanceRecordsToProto(in.Provenance),
	}
}

func ExecutionReportToProto(in replsession.ExecutionReport) *replapiv1.ExecutionReport {
	return &replapiv1.ExecutionReport{Status: in.Status, Result: in.Result, ResultJson: in.ResultJSON, Error: in.Error, DurationMs: in.DurationMS, Awaited: in.Awaited, Console: ConsoleEventsToProto(in.Console), HadSideEffects: in.HadSideFX, HelperError: in.HelperError}
}

func ConsoleEventsToProto(in []replsession.ConsoleEvent) []*replapiv1.ConsoleEvent {
	out := make([]*replapiv1.ConsoleEvent, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.ConsoleEvent{Kind: x.Kind, Message: x.Message})
	}
	return out
}

func StaticReportToProto(in replsession.StaticReport) *replapiv1.StaticReport {
	return &replapiv1.StaticReport{
		Diagnostics:      DiagnosticViewsToProto(in.Diagnostics),
		TopLevelBindings: TopLevelBindingViewsToProto(in.TopLevelBindings),
		Unresolved:       IdentifierUseViewsToProto(in.Unresolved),
		References:       BindingReferenceGroupsToProto(in.References),
		Scope:            ScopeViewToProto(in.Scope),
		Ast:              ASTRowViewsToProto(in.AST),
		AstNodeCount:     uint32(in.ASTNodeCount),
		AstTruncated:     in.ASTTruncated,
		Cst:              CSTNodeViewsToProto(in.CST),
		CstNodeCount:     uint32(in.CSTNodeCount),
		CstTruncated:     in.CSTTruncated,
		FinalExpression:  RangeViewToProto(in.FinalExpression),
		Summary:          StaticSummaryFactsToProto(in.Summary),
	}
}

func RewriteReportToProto(in replsession.RewriteReport) *replapiv1.RewriteReport {
	return &replapiv1.RewriteReport{Mode: in.Mode, DeclaredNames: in.DeclaredNames, HelperNames: in.HelperNames, LastHelperName: in.LastHelperName, BindingHelperName: in.BindingHelperName, CapturedLastExpr: in.CapturedLastExpr, TransformedSource: in.TransformedSource, Operations: RewriteStepsToProto(in.Operations), Warnings: in.Warnings, FinalExpressionSource: in.FinalExpressionSrc}
}

func RuntimeReportToProto(in replsession.RuntimeReport) *replapiv1.RuntimeReport {
	return &replapiv1.RuntimeReport{BeforeGlobals: GlobalStateViewsToProto(in.BeforeGlobals), AfterGlobals: GlobalStateViewsToProto(in.AfterGlobals), Diffs: GlobalDiffViewsToProto(in.Diffs), NewBindings: in.NewBindings, UpdatedBindings: in.UpdatedBindings, RemovedBindings: in.RemovedBindings, LeakedGlobals: in.LeakedGlobals, PersistedByWrap: in.PersistedByWrap, CurrentCellValue: in.CurrentCellValue}
}

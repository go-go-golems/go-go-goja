package pbconv

import (
	replapiv1 "github.com/go-go-golems/go-go-goja/pkg/replapi/pb/proto/goja/replapi/v1"
	"github.com/go-go-golems/go-go-goja/pkg/replsession"
)

func ProvenanceRecordsToProto(in []replsession.ProvenanceRecord) []*replapiv1.ProvenanceRecord {
	out := make([]*replapiv1.ProvenanceRecord, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.ProvenanceRecord{Section: x.Section, Source: x.Source, Notes: x.Notes})
	}
	return out
}

func HistoryEntriesToProto(in []replsession.HistoryEntry) []*replapiv1.HistoryEntry {
	out := make([]*replapiv1.HistoryEntry, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.HistoryEntry{CellId: uint32(x.CellID), CreatedAt: timestamp(x.CreatedAt), SourcePreview: x.SourcePreview, ResultPreview: x.ResultPreview, Status: x.Status})
	}
	return out
}

func BindingViewsToProto(in []replsession.BindingView) []*replapiv1.BindingView {
	out := make([]*replapiv1.BindingView, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.BindingView{Name: x.Name, Kind: x.Kind, Origin: x.Origin, DeclaredInCell: uint32(x.DeclaredInCell), LastUpdatedCell: uint32(x.LastUpdatedCell), DeclaredLine: uint32(x.DeclaredLine), DeclaredSnippet: x.DeclaredSnippet, StaticView: BindingStaticViewToProto(x.Static), Runtime: BindingRuntimeViewToProto(x.Runtime), Provenance: ProvenanceRecordsToProto(x.Provenance)})
	}
	return out
}

func BindingStaticViewToProto(in *replsession.BindingStaticView) *replapiv1.BindingStaticView {
	if in == nil {
		return nil
	}
	return &replapiv1.BindingStaticView{References: IdentifierUseViewsToProto(in.References), Parameters: in.Parameters, Extends: in.Extends, Members: MemberViewsToProto(in.Members)}
}

func BindingRuntimeViewToProto(in replsession.BindingRuntimeView) *replapiv1.BindingRuntimeView {
	return &replapiv1.BindingRuntimeView{ValueKind: in.ValueKind, Preview: in.Preview, OwnProperties: PropertyViewsToProto(in.OwnProperties), PrototypeChain: PrototypeLevelViewsToProto(in.PrototypeChain), FunctionMapping: FunctionMappingViewToProto(in.FunctionMapping)}
}

func PrototypeLevelViewsToProto(in []replsession.PrototypeLevelView) []*replapiv1.PrototypeLevelView {
	out := make([]*replapiv1.PrototypeLevelView, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.PrototypeLevelView{Name: x.Name, Properties: PropertyViewsToProto(x.Properties)})
	}
	return out
}

func PropertyViewsToProto(in []replsession.PropertyView) []*replapiv1.PropertyView {
	out := make([]*replapiv1.PropertyView, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.PropertyView{Name: x.Name, Kind: x.Kind, Preview: x.Preview, IsSymbol: x.IsSymbol, Descriptor_: DescriptorViewToProto(x.Descriptor)})
	}
	return out
}

func DescriptorViewToProto(in *replsession.DescriptorView) *replapiv1.DescriptorView {
	if in == nil {
		return nil
	}
	return &replapiv1.DescriptorView{Writable: in.Writable, Enumerable: in.Enumerable, Configurable: in.Configurable, HasGetter: in.HasGetter, HasSetter: in.HasSetter}
}

func FunctionMappingViewToProto(in *replsession.FunctionMappingView) *replapiv1.FunctionMappingView {
	if in == nil {
		return nil
	}
	return &replapiv1.FunctionMappingView{Name: in.Name, ClassName: in.ClassName, StartLine: uint32(in.StartLine), StartCol: uint32(in.StartCol), EndLine: uint32(in.EndLine), EndCol: uint32(in.EndCol), NodeId: uint32(in.NodeID)}
}

func GlobalStateViewsToProto(in []replsession.GlobalStateView) []*replapiv1.GlobalStateView {
	out := make([]*replapiv1.GlobalStateView, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.GlobalStateView{Name: x.Name, Kind: x.Kind, Preview: x.Preview, Identity: x.Identity, PropertyCount: uint32(x.PropertyCount)})
	}
	return out
}

func GlobalDiffViewsToProto(in []replsession.GlobalDiffView) []*replapiv1.GlobalDiffView {
	out := make([]*replapiv1.GlobalDiffView, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.GlobalDiffView{Name: x.Name, Change: x.Change, Before: x.Before, After: x.After, BeforeKind: x.BeforeKind, AfterKind: x.AfterKind, SessionBound: x.SessionBound})
	}
	return out
}

func DiagnosticViewsToProto(in []replsession.DiagnosticView) []*replapiv1.DiagnosticView {
	out := make([]*replapiv1.DiagnosticView, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.DiagnosticView{Severity: x.Severity, Message: x.Message})
	}
	return out
}

func TopLevelBindingViewsToProto(in []replsession.TopLevelBindingView) []*replapiv1.TopLevelBindingView {
	out := make([]*replapiv1.TopLevelBindingView, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.TopLevelBindingView{Name: x.Name, Kind: x.Kind, Line: uint32(x.Line), Snippet: x.Snippet, Extends: x.Extends, ReferenceCount: uint32(x.ReferenceCount)})
	}
	return out
}

func BindingReferenceGroupsToProto(in []replsession.BindingReferenceGroup) []*replapiv1.BindingReferenceGroup {
	out := make([]*replapiv1.BindingReferenceGroup, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.BindingReferenceGroup{Name: x.Name, Kind: x.Kind, Locations: IdentifierUseViewsToProto(x.Locations)})
	}
	return out
}

func IdentifierUseViewsToProto(in []replsession.IdentifierUseView) []*replapiv1.IdentifierUseView {
	out := make([]*replapiv1.IdentifierUseView, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.IdentifierUseView{Line: uint32(x.Line), Col: uint32(x.Col), Context: x.Context, NodeId: uint32(x.NodeID), Snippet: x.Snippet})
	}
	return out
}

func ScopeViewToProto(in *replsession.ScopeView) *replapiv1.ScopeView {
	if in == nil {
		return nil
	}
	children := make([]*replapiv1.ScopeView, 0, len(in.Children))
	for _, child := range in.Children {
		children = append(children, ScopeViewToProto(child))
	}
	return &replapiv1.ScopeView{Id: uint32(in.ID), Kind: in.Kind, Start: uint32(in.Start), End: uint32(in.End), Bindings: ScopeBindingsToProto(in.Bindings), Children: children}
}

func ScopeBindingsToProto(in []replsession.ScopeBinding) []*replapiv1.ScopeBinding {
	out := make([]*replapiv1.ScopeBinding, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.ScopeBinding{Name: x.Name, Kind: x.Kind})
	}
	return out
}

func ASTRowViewsToProto(in []replsession.ASTRowView) []*replapiv1.ASTRowView {
	out := make([]*replapiv1.ASTRowView, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.ASTRowView{NodeId: uint32(x.NodeID), Title: x.Title, Description: x.Description})
	}
	return out
}

func CSTNodeViewsToProto(in []replsession.CSTNodeView) []*replapiv1.CSTNodeView {
	out := make([]*replapiv1.CSTNodeView, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.CSTNodeView{Depth: uint32(x.Depth), Kind: x.Kind, Text: x.Text, StartRow: uint32(x.StartRow), StartCol: uint32(x.StartCol), EndRow: uint32(x.EndRow), EndCol: uint32(x.EndCol), IsError: x.IsError, IsMissing: x.IsMissing})
	}
	return out
}

func RangeViewToProto(in *replsession.RangeView) *replapiv1.RangeView {
	if in == nil {
		return nil
	}
	return &replapiv1.RangeView{StartLine: uint32(in.StartLine), StartCol: uint32(in.StartCol), EndLine: uint32(in.EndLine), EndCol: uint32(in.EndCol)}
}

func MemberViewsToProto(in []replsession.MemberView) []*replapiv1.MemberView {
	out := make([]*replapiv1.MemberView, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.MemberView{Name: x.Name, Kind: x.Kind, Preview: x.Preview, Inherited: x.Inherited, Source: x.Source})
	}
	return out
}

func StaticSummaryFactsToProto(in []replsession.StaticSummaryFact) []*replapiv1.StaticSummaryFact {
	out := make([]*replapiv1.StaticSummaryFact, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.StaticSummaryFact{Label: x.Label, Value: x.Value})
	}
	return out
}

func RewriteStepsToProto(in []replsession.RewriteStep) []*replapiv1.RewriteStep {
	out := make([]*replapiv1.RewriteStep, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.RewriteStep{Kind: x.Kind, Detail: x.Detail})
	}
	return out
}

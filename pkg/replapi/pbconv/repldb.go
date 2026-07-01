package pbconv

import (
	"fmt"

	replapiv1 "github.com/go-go-golems/go-go-goja/pkg/replapi/pb/proto/goja/replapi/v1"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
)

func ListSessionsResponseToProto(in []repldb.SessionRecord) (*replapiv1.ListSessionsResponse, error) {
	out := &replapiv1.ListSessionsResponse{SchemaVersion: SchemaVersion, Sessions: make([]*replapiv1.SessionRecord, 0, len(in))}
	for _, record := range in {
		pb, err := SessionRecordToProto(record)
		if err != nil {
			return nil, err
		}
		out.Sessions = append(out.Sessions, pb)
	}
	return out, nil
}

func SessionRecordToProto(in repldb.SessionRecord) (*replapiv1.SessionRecord, error) {
	metadata, err := RawJSONToValue(in.MetadataJSON)
	if err != nil {
		return nil, fmt.Errorf("session %s metadata: %w", in.SessionID, err)
	}
	return &replapiv1.SessionRecord{SessionId: in.SessionID, CreatedAt: timestamp(in.CreatedAt), UpdatedAt: timestamp(in.UpdatedAt), DeletedAt: timestampPtr(in.DeletedAt), EngineKind: in.EngineKind, MetadataJson: metadata}, nil
}

func SessionExportToProto(in *repldb.SessionExport) (*replapiv1.SessionExport, error) {
	if in == nil {
		return nil, nil
	}
	session, err := SessionRecordToProto(in.Session)
	if err != nil {
		return nil, err
	}
	out := &replapiv1.SessionExport{Session: session, Evaluations: make([]*replapiv1.EvaluationRecord, 0, len(in.Evaluations))}
	for _, record := range in.Evaluations {
		pb, err := EvaluationRecordToProto(record)
		if err != nil {
			return nil, err
		}
		out.Evaluations = append(out.Evaluations, pb)
	}
	return out, nil
}

func EvaluationRecordsToProto(in []repldb.EvaluationRecord) ([]*replapiv1.EvaluationRecord, error) {
	out := make([]*replapiv1.EvaluationRecord, 0, len(in))
	for _, record := range in {
		pb, err := EvaluationRecordToProto(record)
		if err != nil {
			return nil, err
		}
		out = append(out, pb)
	}
	return out, nil
}

func EvaluationRecordToProto(in repldb.EvaluationRecord) (*replapiv1.EvaluationRecord, error) {
	resultJSON, err := RawJSONToValue(in.ResultJSON)
	if err != nil {
		return nil, fmt.Errorf("evaluation %d result JSON: %w", in.EvaluationID, err)
	}
	analysisJSON, err := RawJSONToValue(in.AnalysisJSON)
	if err != nil {
		return nil, fmt.Errorf("evaluation %d analysis JSON: %w", in.EvaluationID, err)
	}
	globalsBeforeJSON, err := RawJSONToValue(in.GlobalsBeforeJSON)
	if err != nil {
		return nil, fmt.Errorf("evaluation %d globals before JSON: %w", in.EvaluationID, err)
	}
	globalsAfterJSON, err := RawJSONToValue(in.GlobalsAfterJSON)
	if err != nil {
		return nil, fmt.Errorf("evaluation %d globals after JSON: %w", in.EvaluationID, err)
	}
	bindingVersions, err := BindingVersionRecordsToProto(in.BindingVersions)
	if err != nil {
		return nil, err
	}
	bindingDocs, err := BindingDocRecordsToProto(in.BindingDocs)
	if err != nil {
		return nil, err
	}
	return &replapiv1.EvaluationRecord{EvaluationId: in.EvaluationID, SessionId: in.SessionID, CellId: uint32FromInt(in.CellID), CreatedAt: timestamp(in.CreatedAt), RawSource: in.RawSource, RewrittenSource: in.RewrittenSource, Ok: in.OK, ResultJson: resultJSON, ErrorText: in.ErrorText, AnalysisJson: analysisJSON, GlobalsBeforeJson: globalsBeforeJSON, GlobalsAfterJson: globalsAfterJSON, ConsoleEvents: ConsoleEventRecordsToProto(in.ConsoleEvents), BindingVersions: bindingVersions, BindingDocs: bindingDocs}, nil
}

func ConsoleEventRecordsToProto(in []repldb.ConsoleEventRecord) []*replapiv1.ConsoleEventRecord {
	out := make([]*replapiv1.ConsoleEventRecord, 0, len(in))
	for _, x := range in {
		out = append(out, &replapiv1.ConsoleEventRecord{Stream: x.Stream, Seq: uint32FromInt(x.Seq), Text: x.Text})
	}
	return out
}

func BindingVersionRecordsToProto(in []repldb.BindingVersionRecord) ([]*replapiv1.BindingVersionRecord, error) {
	out := make([]*replapiv1.BindingVersionRecord, 0, len(in))
	for _, record := range in {
		summary, err := RawJSONToValue(record.SummaryJSON)
		if err != nil {
			return nil, fmt.Errorf("binding version %s summary JSON: %w", record.Name, err)
		}
		exportJSON, err := RawJSONToValue(record.ExportJSON)
		if err != nil {
			return nil, fmt.Errorf("binding version %s export JSON: %w", record.Name, err)
		}
		out = append(out, &replapiv1.BindingVersionRecord{Name: record.Name, CreatedAt: timestamp(record.CreatedAt), CellId: uint32FromInt(record.CellID), Action: record.Action, RuntimeType: record.RuntimeType, DisplayValue: record.DisplayValue, SummaryJson: summary, ExportKind: record.ExportKind, ExportJson: exportJSON, DocDigest: record.DocDigest})
	}
	return out, nil
}

func BindingDocRecordsToProto(in []repldb.BindingDocRecord) ([]*replapiv1.BindingDocRecord, error) {
	out := make([]*replapiv1.BindingDocRecord, 0, len(in))
	for _, record := range in {
		normalized, err := RawJSONToValue(record.NormalizedJSON)
		if err != nil {
			return nil, fmt.Errorf("binding doc %s normalized JSON: %w", record.SymbolName, err)
		}
		out = append(out, &replapiv1.BindingDocRecord{SymbolName: record.SymbolName, CellId: uint32FromInt(record.CellID), SourceKind: record.SourceKind, RawDoc: record.RawDoc, NormalizedJson: normalized})
	}
	return out, nil
}

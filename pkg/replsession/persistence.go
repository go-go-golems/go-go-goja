package replsession

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dop251/goja"
	inspectorruntime "github.com/go-go-golems/go-go-goja/pkg/inspector/runtime"
	"github.com/go-go-golems/go-go-goja/pkg/jsdoc/extract"
	"github.com/go-go-golems/go-go-goja/pkg/repldb"
	"github.com/pkg/errors"
)

func (s *Service) persistCell(ctx context.Context, state *sessionState, cell *CellReport) error {
	if s.store == nil || state == nil || !state.policy.Persist.Enabled || !state.policy.Persist.Evaluations {
		return nil
	}
	if state == nil || cell == nil {
		return errors.New("persist cell: state or cell is nil")
	}

	resultJSON, err := json.Marshal(cell)
	if err != nil {
		return errors.Wrap(err, "persist cell: marshal cell report")
	}
	analysisJSON, err := json.Marshal(cell.Static)
	if err != nil {
		return errors.Wrap(err, "persist cell: marshal static report")
	}
	globalsBeforeJSON, err := json.Marshal(cell.Runtime.BeforeGlobals)
	if err != nil {
		return errors.Wrap(err, "persist cell: marshal globals before")
	}
	globalsAfterJSON, err := json.Marshal(cell.Runtime.AfterGlobals)
	if err != nil {
		return errors.Wrap(err, "persist cell: marshal globals after")
	}

	consoleEvents := make([]repldb.ConsoleEventRecord, 0, len(cell.Execution.Console))
	for idx, event := range cell.Execution.Console {
		consoleEvents = append(consoleEvents, repldb.ConsoleEventRecord{
			Stream: event.Kind,
			Seq:    idx + 1,
			Text:   event.Message,
		})
	}
	bindingVersions, bindingDocs, err := s.bindingPersistenceRecords(ctx, state, cell)
	if err != nil {
		return err
	}

	if err := s.store.PersistEvaluation(ctx, repldb.EvaluationRecord{
		SessionID:         state.id,
		CellID:            cell.ID,
		CreatedAt:         cell.CreatedAt,
		RawSource:         cell.Source,
		RewrittenSource:   cell.Rewrite.TransformedSource,
		OK:                cell.Execution.Status == "ok",
		ResultJSON:        resultJSON,
		ErrorText:         cell.Execution.Error,
		AnalysisJSON:      analysisJSON,
		GlobalsBeforeJSON: globalsBeforeJSON,
		GlobalsAfterJSON:  globalsAfterJSON,
		ConsoleEvents:     consoleEvents,
		BindingVersions:   bindingVersions,
		BindingDocs:       bindingDocs,
	}); err != nil {
		return errors.Wrap(err, "persist cell: write evaluation")
	}

	return nil
}

func (s *Service) bindingPersistenceRecords(ctx context.Context, state *sessionState, cell *CellReport) ([]repldb.BindingVersionRecord, []repldb.BindingDocRecord, error) {
	docRecords := []repldb.BindingDocRecord{}
	docDigests := map[string]string{}
	var err error
	if state.policy.Persist.BindingDocs && state.policy.Observe.JSDocExtraction {
		docRecords, docDigests, err = extractBindingDocs(cell)
		if err != nil {
			return nil, nil, err
		}
	}

	if !state.policy.Persist.BindingVersions {
		return nil, docRecords, nil
	}

	changedNames := append([]string(nil), cell.Runtime.NewBindings...)
	changedNames = append(changedNames, cell.Runtime.UpdatedBindings...)
	exportSnapshots, err := state.snapshotBindingExports(ctx, changedNames)
	if err != nil {
		return nil, nil, errors.Wrap(err, "persist cell: snapshot binding exports")
	}

	versionRecords := make([]repldb.BindingVersionRecord, 0, len(changedNames)+len(cell.Runtime.RemovedBindings))
	for _, name := range dedupeSortedStrings(cell.Runtime.NewBindings) {
		record, ok, err := state.bindingVersionRecord(name, cell.ID, cell.CreatedAt, "insert", exportSnapshots[name], docDigests[name])
		if err != nil {
			return nil, nil, err
		}
		if ok {
			versionRecords = append(versionRecords, record)
		}
	}
	for _, name := range dedupeSortedStrings(cell.Runtime.UpdatedBindings) {
		record, ok, err := state.bindingVersionRecord(name, cell.ID, cell.CreatedAt, "update", exportSnapshots[name], docDigests[name])
		if err != nil {
			return nil, nil, err
		}
		if ok {
			versionRecords = append(versionRecords, record)
		}
	}
	for _, name := range dedupeSortedStrings(cell.Runtime.RemovedBindings) {
		record, ok, err := bindingRemovalRecord(cell, name, docDigests[name])
		if err != nil {
			return nil, nil, err
		}
		if ok {
			versionRecords = append(versionRecords, record)
		}
	}

	return versionRecords, docRecords, nil
}

type bindingExportSnapshot struct {
	ExportKind string
	ExportJSON string
}

func (s *sessionState) snapshotBindingExports(ctx context.Context, names []string) (map[string]bindingExportSnapshot, error) {
	names = dedupeSortedStrings(names)
	if len(names) == 0 {
		return map[string]bindingExportSnapshot{}, nil
	}

	ret, err := s.runtime.Owner.Call(ctx, "replsession.snapshot-binding-exports", func(_ context.Context, vm *goja.Runtime) (any, error) {
		out := make(map[string]bindingExportSnapshot, len(names))
		for _, name := range names {
			out[name] = classifyBindingExport(vm.Get(name), vm)
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	snapshots, ok := ret.(map[string]bindingExportSnapshot)
	if !ok {
		return nil, fmt.Errorf("unexpected binding export snapshot type %T", ret)
	}
	return snapshots, nil
}

func classifyBindingExport(value goja.Value, vm *goja.Runtime) bindingExportSnapshot {
	if value == nil || goja.IsUndefined(value) {
		return stringExportSnapshot("undefined")
	}
	if goja.IsNull(value) {
		return bindingExportSnapshot{ExportKind: "json", ExportJSON: "null"}
	}
	if _, ok := goja.AssertFunction(value); ok {
		return stringExportSnapshot(inspectorruntime.ValuePreview(value, vm, 120))
	}

	exported := value.Export()
	bytes, err := json.Marshal(exported)
	if err == nil {
		return bindingExportSnapshot{ExportKind: "json", ExportJSON: string(bytes)}
	}
	return stringExportSnapshot(inspectorruntime.ValuePreview(value, vm, 120))
}

func stringExportSnapshot(preview string) bindingExportSnapshot {
	bytes, err := json.Marshal(preview)
	if err != nil {
		return bindingExportSnapshot{ExportKind: "none", ExportJSON: "null"}
	}
	return bindingExportSnapshot{ExportKind: "string", ExportJSON: string(bytes)}
}

func (s *sessionState) bindingVersionRecord(name string, cellID int, createdAt time.Time, action string, exportSnapshot bindingExportSnapshot, docDigest string) (repldb.BindingVersionRecord, bool, error) {
	binding := s.bindings[name]
	if binding == nil || s.isIgnoredGlobal(name) {
		return repldb.BindingVersionRecord{}, false, nil
	}

	summaryJSON, err := json.Marshal(bindingViewFromState(binding))
	if err != nil {
		return repldb.BindingVersionRecord{}, false, errors.Wrap(err, "persist cell: marshal binding summary")
	}

	return repldb.BindingVersionRecord{
		Name:         name,
		CreatedAt:    createdAt,
		CellID:       cellID,
		Action:       action,
		RuntimeType:  binding.Runtime.ValueKind,
		DisplayValue: binding.Runtime.Preview,
		SummaryJSON:  summaryJSON,
		ExportKind:   defaultBindingExportKind(exportSnapshot.ExportKind),
		ExportJSON:   json.RawMessage(defaultBindingExportJSON(exportSnapshot.ExportJSON)),
		DocDigest:    docDigest,
	}, true, nil
}

func bindingRemovalRecord(cell *CellReport, name string, docDigest string) (repldb.BindingVersionRecord, bool, error) {
	if cell == nil {
		return repldb.BindingVersionRecord{}, false, nil
	}
	for _, diff := range cell.Runtime.Diffs {
		if diff.Name != name || diff.Change != "removed" {
			continue
		}
		summaryJSON, err := json.Marshal(diff)
		if err != nil {
			return repldb.BindingVersionRecord{}, false, errors.Wrap(err, "persist cell: marshal removal summary")
		}
		return repldb.BindingVersionRecord{
			Name:         name,
			CreatedAt:    cell.CreatedAt,
			CellID:       cell.ID,
			Action:       "remove",
			RuntimeType:  diff.BeforeKind,
			DisplayValue: diff.Before,
			SummaryJSON:  summaryJSON,
			ExportKind:   "none",
			ExportJSON:   json.RawMessage(`null`),
			DocDigest:    docDigest,
		}, true, nil
	}
	return repldb.BindingVersionRecord{}, false, nil
}

func extractBindingDocs(cell *CellReport) ([]repldb.BindingDocRecord, map[string]string, error) {
	if cell == nil || cell.Execution.Status == "parse-error" {
		return nil, map[string]string{}, nil
	}

	fileDoc, err := extract.ParseSource(fmt.Sprintf("<repl-cell-%d>", cell.ID), []byte(cell.Source))
	if err != nil {
		return nil, nil, errors.Wrap(err, "persist cell: extract jsdocex docs")
	}

	docRecords := make([]repldb.BindingDocRecord, 0, len(fileDoc.Symbols))
	docPayloads := map[string][]string{}
	for _, symbol := range fileDoc.Symbols {
		if symbol == nil || strings.TrimSpace(symbol.Name) == "" {
			continue
		}
		normalizedJSON, err := json.Marshal(symbol)
		if err != nil {
			return nil, nil, errors.Wrap(err, "persist cell: marshal symbol doc")
		}
		name := strings.TrimSpace(symbol.Name)
		docRecords = append(docRecords, repldb.BindingDocRecord{
			SymbolName:     name,
			CellID:         cell.ID,
			SourceKind:     "jsdocex",
			RawDoc:         string(normalizedJSON),
			NormalizedJSON: normalizedJSON,
		})
		docPayloads[name] = append(docPayloads[name], string(normalizedJSON))
	}

	digests := make(map[string]string, len(docPayloads))
	for name, payloads := range docPayloads {
		sort.Strings(payloads)
		h := sha256.New()
		for _, payload := range payloads {
			_, _ = h.Write([]byte(payload))
			_, _ = h.Write([]byte{'\n'})
		}
		digests[name] = hex.EncodeToString(h.Sum(nil))
	}

	return docRecords, digests, nil
}

func defaultBindingExportKind(kind string) string {
	if strings.TrimSpace(kind) == "" {
		return "none"
	}
	return kind
}

func defaultBindingExportJSON(value string) string {
	if strings.TrimSpace(value) == "" {
		return "null"
	}
	return value
}

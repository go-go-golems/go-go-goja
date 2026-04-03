package repldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// LoadSession returns the durable session record for sessionID.
func (s *Store) LoadSession(ctx context.Context, sessionID string) (SessionRecord, error) {
	if s == nil || s.db == nil {
		return SessionRecord{}, errors.New("load session: store is nil")
	}
	if strings.TrimSpace(sessionID) == "" {
		return SessionRecord{}, errors.New("load session: session id is empty")
	}
	if err := ctx.Err(); err != nil {
		return SessionRecord{}, err
	}

	var (
		record            SessionRecord
		createdAtRaw      string
		updatedAtRaw      string
		deletedAtNullable sql.NullString
		metadataJSON      string
	)
	err := s.db.QueryRowContext(
		ctx,
		`SELECT session_id, created_at, updated_at, deleted_at, engine_kind, metadata_json
		 FROM sessions
		 WHERE session_id = ?`,
		sessionID,
	).Scan(
		&record.SessionID,
		&createdAtRaw,
		&updatedAtRaw,
		&deletedAtNullable,
		&record.EngineKind,
		&metadataJSON,
	)
	if err != nil {
		return SessionRecord{}, errors.Wrap(err, "load session")
	}

	record.CreatedAt = parseTime(createdAtRaw)
	record.UpdatedAt = parseTime(updatedAtRaw)
	if deletedAtNullable.Valid {
		deletedAt := parseTime(deletedAtNullable.String)
		record.DeletedAt = &deletedAt
	}
	record.MetadataJSON = json.RawMessage(metadataJSON)
	return record, nil
}

// LoadEvaluations returns all persisted evaluations for sessionID in cell order.
func (s *Store) LoadEvaluations(ctx context.Context, sessionID string) ([]EvaluationRecord, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("load evaluations: store is nil")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("load evaluations: session id is empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT evaluation_id, session_id, cell_id, created_at, raw_source, rewritten_source,
		        ok, result_json, error_text, analysis_json, globals_before_json, globals_after_json
		 FROM evaluations
		 WHERE session_id = ?
		 ORDER BY cell_id ASC`,
		sessionID,
	)
	if err != nil {
		return nil, errors.Wrap(err, "load evaluations")
	}
	defer func() { _ = rows.Close() }()

	records := []EvaluationRecord{}
	for rows.Next() {
		var (
			record            EvaluationRecord
			createdAtRaw      string
			okValue           int
			resultJSON        string
			analysisJSON      string
			globalsBeforeJSON string
			globalsAfterJSON  string
		)
		if err := rows.Scan(
			&record.EvaluationID,
			&record.SessionID,
			&record.CellID,
			&createdAtRaw,
			&record.RawSource,
			&record.RewrittenSource,
			&okValue,
			&resultJSON,
			&record.ErrorText,
			&analysisJSON,
			&globalsBeforeJSON,
			&globalsAfterJSON,
		); err != nil {
			return nil, errors.Wrap(err, "load evaluations: scan row")
		}
		record.CreatedAt = parseTime(createdAtRaw)
		record.OK = okValue != 0
		record.ResultJSON = json.RawMessage(resultJSON)
		record.AnalysisJSON = json.RawMessage(analysisJSON)
		record.GlobalsBeforeJSON = json.RawMessage(globalsBeforeJSON)
		record.GlobalsAfterJSON = json.RawMessage(globalsAfterJSON)

		consoleEvents, err := loadConsoleEvents(ctx, s.db, record.EvaluationID)
		if err != nil {
			return nil, err
		}
		record.ConsoleEvents = consoleEvents

		bindingVersions, err := loadBindingVersions(ctx, s.db, record.EvaluationID)
		if err != nil {
			return nil, err
		}
		record.BindingVersions = bindingVersions

		bindingDocs, err := loadBindingDocs(ctx, s.db, record.EvaluationID)
		if err != nil {
			return nil, err
		}
		record.BindingDocs = bindingDocs

		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "load evaluations: iterate rows")
	}

	return records, nil
}

// ExportSession loads a full structured session export from SQLite.
func (s *Store) ExportSession(ctx context.Context, sessionID string) (*SessionExport, error) {
	session, err := s.LoadSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	evaluations, err := s.LoadEvaluations(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return &SessionExport{
		Session:     session,
		Evaluations: evaluations,
	}, nil
}

// LoadReplaySource returns the raw cell source in evaluation order for replay.
func (s *Store) LoadReplaySource(ctx context.Context, sessionID string) ([]string, error) {
	evaluations, err := s.LoadEvaluations(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(evaluations))
	for _, evaluation := range evaluations {
		out = append(out, evaluation.RawSource)
	}
	return out, nil
}

func loadConsoleEvents(ctx context.Context, db *sql.DB, evaluationID int64) ([]ConsoleEventRecord, error) {
	rows, err := db.QueryContext(
		ctx,
		`SELECT stream, seq, text
		 FROM console_events
		 WHERE evaluation_id = ?
		 ORDER BY seq ASC`,
		evaluationID,
	)
	if err != nil {
		return nil, errors.Wrap(err, "load evaluations: query console events")
	}
	defer func() { _ = rows.Close() }()

	events := []ConsoleEventRecord{}
	for rows.Next() {
		var event ConsoleEventRecord
		if err := rows.Scan(&event.Stream, &event.Seq, &event.Text); err != nil {
			return nil, errors.Wrap(err, "load evaluations: scan console event")
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "load evaluations: iterate console events")
	}
	return events, nil
}

func loadBindingVersions(ctx context.Context, db *sql.DB, evaluationID int64) ([]BindingVersionRecord, error) {
	rows, err := db.QueryContext(
		ctx,
		`SELECT b.name, bv.cell_id, bv.action, bv.runtime_type, bv.display_value, bv.summary_json, bv.export_kind, bv.export_json, bv.doc_digest
		 FROM binding_versions bv
		 JOIN bindings b ON b.binding_id = bv.binding_id
		 WHERE bv.evaluation_id = ?
		 ORDER BY bv.binding_version_id ASC`,
		evaluationID,
	)
	if err != nil {
		return nil, errors.Wrap(err, "load evaluations: query binding versions")
	}
	defer func() { _ = rows.Close() }()

	versions := []BindingVersionRecord{}
	for rows.Next() {
		var (
			record      BindingVersionRecord
			summaryJSON string
			exportJSON  string
		)
		if err := rows.Scan(
			&record.Name,
			&record.CellID,
			&record.Action,
			&record.RuntimeType,
			&record.DisplayValue,
			&summaryJSON,
			&record.ExportKind,
			&exportJSON,
			&record.DocDigest,
		); err != nil {
			return nil, errors.Wrap(err, "load evaluations: scan binding version")
		}
		record.SummaryJSON = json.RawMessage(summaryJSON)
		record.ExportJSON = json.RawMessage(exportJSON)
		versions = append(versions, record)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "load evaluations: iterate binding versions")
	}
	return versions, nil
}

func loadBindingDocs(ctx context.Context, db *sql.DB, evaluationID int64) ([]BindingDocRecord, error) {
	rows, err := db.QueryContext(
		ctx,
		`SELECT symbol_name, cell_id, source_kind, raw_doc, normalized_json
		 FROM binding_docs
		 WHERE evaluation_id = ?
		 ORDER BY binding_doc_id ASC`,
		evaluationID,
	)
	if err != nil {
		return nil, errors.Wrap(err, "load evaluations: query binding docs")
	}
	defer func() { _ = rows.Close() }()

	docs := []BindingDocRecord{}
	for rows.Next() {
		var (
			record         BindingDocRecord
			normalizedJSON string
		)
		if err := rows.Scan(
			&record.SymbolName,
			&record.CellID,
			&record.SourceKind,
			&record.RawDoc,
			&normalizedJSON,
		); err != nil {
			return nil, errors.Wrap(err, "load evaluations: scan binding doc")
		}
		record.NormalizedJSON = json.RawMessage(normalizedJSON)
		docs = append(docs, record)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "load evaluations: iterate binding docs")
	}
	return docs, nil
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

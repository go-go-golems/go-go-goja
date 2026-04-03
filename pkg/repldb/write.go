package repldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// CreateSession inserts a durable session row.
func (s *Store) CreateSession(ctx context.Context, record SessionRecord) error {
	if s == nil || s.db == nil {
		return errors.New("create session: store is nil")
	}
	if strings.TrimSpace(record.SessionID) == "" {
		return errors.New("create session: session id is empty")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	createdAt := normalizeTime(record.CreatedAt)
	updatedAt := normalizeTime(record.UpdatedAt)
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	if createdAt.IsZero() {
		createdAt = updatedAt
	}
	engineKind := strings.TrimSpace(record.EngineKind)
	if engineKind == "" {
		engineKind = "goja"
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO sessions(session_id, created_at, updated_at, deleted_at, engine_kind, metadata_json)
		 VALUES(?, ?, ?, ?, ?, ?)`,
		record.SessionID,
		createdAt.Format(time.RFC3339Nano),
		updatedAt.Format(time.RFC3339Nano),
		formatNullableTime(record.DeletedAt),
		engineKind,
		jsonOrDefault(record.MetadataJSON, `{}`),
	)
	if err != nil {
		return errors.Wrap(err, "create session")
	}

	return nil
}

// DeleteSession records durable deletion metadata for a session.
func (s *Store) DeleteSession(ctx context.Context, sessionID string, deletedAt time.Time) error {
	if s == nil || s.db == nil {
		return errors.New("delete session: store is nil")
	}
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("delete session: session id is empty")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	deletedAt = normalizeTime(deletedAt)
	if deletedAt.IsZero() {
		deletedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE sessions
		 SET deleted_at = ?, updated_at = ?
		 WHERE session_id = ?`,
		deletedAt.Format(time.RFC3339Nano),
		deletedAt.Format(time.RFC3339Nano),
		sessionID,
	)
	if err != nil {
		return errors.Wrap(err, "delete session")
	}

	return nil
}

// PersistEvaluation writes an evaluation and its child rows in one transaction.
func (s *Store) PersistEvaluation(ctx context.Context, record EvaluationRecord) error {
	if s == nil || s.db == nil {
		return errors.New("persist evaluation: store is nil")
	}
	if strings.TrimSpace(record.SessionID) == "" {
		return errors.New("persist evaluation: session id is empty")
	}
	if record.CellID <= 0 {
		return errors.New("persist evaluation: cell id must be positive")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "persist evaluation: begin tx")
	}
	defer func() { _ = tx.Rollback() }()

	evaluationID, err := insertEvaluationTx(ctx, tx, record)
	if err != nil {
		return err
	}
	if err := insertConsoleEventsTx(ctx, tx, evaluationID, record.ConsoleEvents); err != nil {
		return err
	}
	if err := insertBindingVersionsTx(ctx, tx, record, evaluationID); err != nil {
		return err
	}
	if err := insertBindingDocsTx(ctx, tx, record, evaluationID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE sessions
		 SET updated_at = ?
		 WHERE session_id = ?`,
		normalizeTime(record.CreatedAt).Format(time.RFC3339Nano),
		record.SessionID,
	); err != nil {
		return errors.Wrap(err, "persist evaluation: update session timestamp")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "persist evaluation: commit tx")
	}

	return nil
}

func insertEvaluationTx(ctx context.Context, tx *sql.Tx, record EvaluationRecord) (int64, error) {
	result, err := tx.ExecContext(
		ctx,
		`INSERT INTO evaluations(
			session_id, cell_id, created_at, raw_source, rewritten_source,
			ok, result_json, error_text, analysis_json, globals_before_json, globals_after_json
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.SessionID,
		record.CellID,
		normalizeTime(record.CreatedAt).Format(time.RFC3339Nano),
		record.RawSource,
		record.RewrittenSource,
		boolToInt(record.OK),
		jsonOrDefault(record.ResultJSON, `{}`),
		record.ErrorText,
		jsonOrDefault(record.AnalysisJSON, `{}`),
		jsonOrDefault(record.GlobalsBeforeJSON, `[]`),
		jsonOrDefault(record.GlobalsAfterJSON, `[]`),
	)
	if err != nil {
		return 0, errors.Wrap(err, "persist evaluation: insert evaluation")
	}

	evaluationID, err := result.LastInsertId()
	if err != nil {
		return 0, errors.Wrap(err, "persist evaluation: read evaluation id")
	}
	return evaluationID, nil
}

func insertConsoleEventsTx(ctx context.Context, tx *sql.Tx, evaluationID int64, events []ConsoleEventRecord) error {
	if len(events) == 0 {
		return nil
	}

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO console_events(evaluation_id, stream, seq, text) VALUES(?, ?, ?, ?)`)
	if err != nil {
		return errors.Wrap(err, "persist evaluation: prepare console events")
	}
	defer func() { _ = stmt.Close() }()

	for idx, event := range events {
		seq := event.Seq
		if seq == 0 {
			seq = idx + 1
		}
		if _, err := stmt.ExecContext(ctx, evaluationID, strings.TrimSpace(event.Stream), seq, event.Text); err != nil {
			return errors.Wrap(err, "persist evaluation: insert console event")
		}
	}
	return nil
}

func insertBindingVersionsTx(ctx context.Context, tx *sql.Tx, record EvaluationRecord, evaluationID int64) error {
	for _, version := range record.BindingVersions {
		bindingID, err := ensureBindingTx(ctx, tx, record.SessionID, version.Name, version.CreatedAt, evaluationID, record.CellID)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO binding_versions(
				binding_id, evaluation_id, cell_id, action, runtime_type, display_value,
				summary_json, export_kind, export_json, doc_digest
			) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			bindingID,
			evaluationID,
			record.CellID,
			defaultString(version.Action, "upsert"),
			defaultString(version.RuntimeType, "unknown"),
			version.DisplayValue,
			jsonOrDefault(version.SummaryJSON, `{}`),
			defaultString(version.ExportKind, "none"),
			jsonOrDefault(version.ExportJSON, `null`),
			version.DocDigest,
		); err != nil {
			return errors.Wrap(err, "persist evaluation: insert binding version")
		}
		if _, err := tx.ExecContext(
			ctx,
			`UPDATE bindings
			 SET latest_evaluation_id = ?, latest_cell_id = ?
			 WHERE binding_id = ?`,
			evaluationID,
			record.CellID,
			bindingID,
		); err != nil {
			return errors.Wrap(err, "persist evaluation: update binding latest pointers")
		}
	}
	return nil
}

func insertBindingDocsTx(ctx context.Context, tx *sql.Tx, record EvaluationRecord, evaluationID int64) error {
	for _, doc := range record.BindingDocs {
		bindingID, err := ensureBindingTx(ctx, tx, record.SessionID, doc.SymbolName, record.CreatedAt, evaluationID, record.CellID)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO binding_docs(
				binding_id, evaluation_id, cell_id, symbol_name, source_kind, raw_doc, normalized_json
			) VALUES(?, ?, ?, ?, ?, ?, ?)`,
			bindingID,
			evaluationID,
			doc.CellID,
			doc.SymbolName,
			defaultString(doc.SourceKind, "repl-source"),
			doc.RawDoc,
			jsonOrDefault(doc.NormalizedJSON, `{}`),
		); err != nil {
			return errors.Wrap(err, "persist evaluation: insert binding doc")
		}
	}
	return nil
}

func ensureBindingTx(ctx context.Context, tx *sql.Tx, sessionID string, name string, createdAt time.Time, evaluationID int64, cellID int) (int64, error) {
	normalizedName := strings.TrimSpace(name)
	if normalizedName == "" {
		return 0, errors.New("persist evaluation: binding name is empty")
	}

	result, err := tx.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO bindings(session_id, name, created_at, latest_evaluation_id, latest_cell_id)
		 VALUES(?, ?, ?, ?, ?)`,
		sessionID,
		normalizedName,
		normalizeTime(createdAt).Format(time.RFC3339Nano),
		evaluationID,
		cellID,
	)
	if err != nil {
		return 0, errors.Wrap(err, "persist evaluation: insert binding")
	}

	if rows, rowsErr := result.RowsAffected(); rowsErr == nil && rows > 0 {
		id, err := result.LastInsertId()
		if err != nil {
			return 0, errors.Wrap(err, "persist evaluation: read binding id")
		}
		return id, nil
	}

	var bindingID int64
	if err := tx.QueryRowContext(
		ctx,
		`SELECT binding_id FROM bindings WHERE session_id = ? AND name = ?`,
		sessionID,
		normalizedName,
	).Scan(&bindingID); err != nil {
		return 0, errors.Wrap(err, "persist evaluation: select binding id")
	}
	return bindingID, nil
}

func normalizeTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t.UTC()
}

func formatNullableTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func jsonOrDefault(value json.RawMessage, fallback string) string {
	if len(value) == 0 {
		return fallback
	}
	return string(value)
}

func defaultString(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

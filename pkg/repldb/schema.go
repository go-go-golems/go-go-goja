package repldb

func schemaStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS repldb_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			session_id TEXT PRIMARY KEY,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			deleted_at TEXT,
			engine_kind TEXT NOT NULL,
			metadata_json TEXT NOT NULL DEFAULT '{}'
		);`,
		`CREATE TABLE IF NOT EXISTS evaluations (
			evaluation_id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			cell_id INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			raw_source TEXT NOT NULL,
			rewritten_source TEXT NOT NULL,
			ok INTEGER NOT NULL,
			result_json TEXT NOT NULL,
			error_text TEXT NOT NULL DEFAULT '',
			analysis_json TEXT NOT NULL,
			globals_before_json TEXT NOT NULL,
			globals_after_json TEXT NOT NULL,
			FOREIGN KEY(session_id) REFERENCES sessions(session_id),
			UNIQUE(session_id, cell_id)
		);`,
		`CREATE TABLE IF NOT EXISTS console_events (
			console_event_id INTEGER PRIMARY KEY AUTOINCREMENT,
			evaluation_id INTEGER NOT NULL,
			stream TEXT NOT NULL,
			seq INTEGER NOT NULL,
			text TEXT NOT NULL,
			FOREIGN KEY(evaluation_id) REFERENCES evaluations(evaluation_id),
			UNIQUE(evaluation_id, seq)
		);`,
		`CREATE TABLE IF NOT EXISTS bindings (
			binding_id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL,
			latest_evaluation_id INTEGER,
			latest_cell_id INTEGER,
			UNIQUE(session_id, name),
			FOREIGN KEY(session_id) REFERENCES sessions(session_id),
			FOREIGN KEY(latest_evaluation_id) REFERENCES evaluations(evaluation_id)
		);`,
		`CREATE TABLE IF NOT EXISTS binding_versions (
			binding_version_id INTEGER PRIMARY KEY AUTOINCREMENT,
			binding_id INTEGER NOT NULL,
			evaluation_id INTEGER NOT NULL,
			cell_id INTEGER NOT NULL,
			action TEXT NOT NULL,
			runtime_type TEXT NOT NULL,
			display_value TEXT NOT NULL,
			summary_json TEXT NOT NULL,
			export_kind TEXT NOT NULL,
			export_json TEXT NOT NULL,
			doc_digest TEXT NOT NULL DEFAULT '',
			FOREIGN KEY(binding_id) REFERENCES bindings(binding_id),
			FOREIGN KEY(evaluation_id) REFERENCES evaluations(evaluation_id),
			UNIQUE(binding_id, evaluation_id)
		);`,
		`CREATE TABLE IF NOT EXISTS binding_docs (
			binding_doc_id INTEGER PRIMARY KEY AUTOINCREMENT,
			binding_id INTEGER NOT NULL,
			evaluation_id INTEGER NOT NULL,
			cell_id INTEGER NOT NULL,
			symbol_name TEXT NOT NULL,
			source_kind TEXT NOT NULL,
			raw_doc TEXT NOT NULL,
			normalized_json TEXT NOT NULL,
			FOREIGN KEY(binding_id) REFERENCES bindings(binding_id),
			FOREIGN KEY(evaluation_id) REFERENCES evaluations(evaluation_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_evaluations_session_id ON evaluations(session_id);`,
		`CREATE INDEX IF NOT EXISTS idx_console_events_evaluation_id ON console_events(evaluation_id);`,
		`CREATE INDEX IF NOT EXISTS idx_bindings_session_id ON bindings(session_id);`,
		`CREATE INDEX IF NOT EXISTS idx_binding_versions_binding_id ON binding_versions(binding_id);`,
		`CREATE INDEX IF NOT EXISTS idx_binding_versions_evaluation_id ON binding_versions(evaluation_id);`,
		`CREATE INDEX IF NOT EXISTS idx_binding_docs_binding_id ON binding_docs(binding_id);`,
		`CREATE INDEX IF NOT EXISTS idx_binding_docs_evaluation_id ON binding_docs(evaluation_id);`,
	}
}

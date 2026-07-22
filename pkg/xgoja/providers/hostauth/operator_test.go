package hostauth

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestBootstrapAdminCommandReconcilesSQLite(t *testing.T) {
	set, err := newOperatorCommandSet(providerapi.CommandSetContext{})
	if err != nil {
		t.Fatalf("new operator command set: %v", err)
	}
	if len(set.Commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(set.Commands))
	}
	command, ok := set.Commands[0].(*bootstrapAdminCommand)
	if !ok {
		t.Fatalf("command type = %T", set.Commands[0])
	}
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "auth.sqlite")
	dsnPath := filepath.Join(dir, "dsn")
	if err := os.WriteFile(dsnPath, []byte(dbPath+"\n"), 0o600); err != nil {
		t.Fatalf("write DSN: %v", err)
	}
	section, ok := command.Description().Schema.Get(schema.DefaultSlug)
	if !ok {
		t.Fatal("command default section missing")
	}
	updates := map[string]any{
		"db-driver": "sqlite", "db-dsn-file": dsnPath, "apply-schema": true,
		"issuer": "https://idp.example.test", "subject": "admin-subject", "email": "admin@example.test",
		"display-name": "Administrator", "organization-id": "o1", "organization-slug": "primary", "organization-name": "Primary Organization",
	}
	fieldValues := fields.NewFieldValues()
	for _, definition := range section.GetDefinitions().ToList() {
		value := any(nil)
		if definition.Default != nil {
			value = *definition.Default
		}
		if override, exists := updates[definition.Name]; exists {
			value = override
		}
		if value != nil {
			fieldValues.Set(definition.Name, &fields.FieldValue{Definition: definition, Value: value})
		}
	}
	sectionValues, err := values.NewSectionValues(section, values.WithFields(fieldValues))
	if err != nil {
		t.Fatalf("new section values: %v", err)
	}
	parsed := values.New(values.WithSectionValues(schema.DefaultSlug, sectionValues))
	processor := &rowCollector{}
	if err := command.RunIntoGlazeProcessor(context.Background(), parsed, processor); err != nil {
		t.Fatalf("first command run: %v", err)
	}
	if err := command.RunIntoGlazeProcessor(context.Background(), parsed, processor); err != nil {
		t.Fatalf("repeat command run: %v", err)
	}
	if len(processor.rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(processor.rows))
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open result db: %v", err)
	}
	defer func() { _ = db.Close() }()
	var memberships, audits int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_app_memberships WHERE role = 'admin' AND revoked_at IS NULL`).Scan(&memberships); err != nil {
		t.Fatalf("count memberships: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_audit_records WHERE event = 'operator.bootstrap_admin'`).Scan(&audits); err != nil {
		t.Fatalf("count audits: %v", err)
	}
	if memberships != 1 || audits != 2 {
		t.Fatalf("memberships=%d audits=%d", memberships, audits)
	}
}

type rowCollector struct{ rows []types.Row }

var _ middlewares.Processor = (*rowCollector)(nil)

func (c *rowCollector) AddRow(_ context.Context, row types.Row) error {
	c.rows = append(c.rows, row)
	return nil
}

func (c *rowCollector) Close(context.Context) error { return nil }

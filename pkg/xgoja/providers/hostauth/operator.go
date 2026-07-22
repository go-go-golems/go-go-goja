package hostauth

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth/adminbootstrap"
	appauthsql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/appauth/sqlstore"
	auditsql "github.com/go-go-golems/go-go-goja/pkg/gojahttp/auth/audit/sqlstore"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/pkg/errors"
)

const maxDSNFileBytes = 16 * 1024

type bootstrapAdminCommand struct {
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = (*bootstrapAdminCommand)(nil)

type bootstrapAdminSettings struct {
	DBDriver         string `glazed:"db-driver"`
	DBDSNFile        string `glazed:"db-dsn-file"`
	ApplySchema      bool   `glazed:"apply-schema"`
	Issuer           string `glazed:"issuer"`
	Subject          string `glazed:"subject"`
	Email            string `glazed:"email"`
	DisplayName      string `glazed:"display-name"`
	OrganizationID   string `glazed:"organization-id"`
	OrganizationSlug string `glazed:"organization-slug"`
	OrganizationName string `glazed:"organization-name"`
	OperatorID       string `glazed:"operator-id"`
}

func newOperatorCommandSet(providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
	command := &bootstrapAdminCommand{CommandDescription: cmds.NewCommandDescription(
		"bootstrap-admin",
		cmds.WithShort("Reconcile the first application organization administrator"),
		cmds.WithLong(`
Bootstrap-admin is an offline, idempotent deployment operation. It binds one
immutable OIDC issuer/subject identity to the generated application's initial
organization and active admin membership, and records the operation in the
same database transaction.

The database DSN must come from a mounted file so credentials are not exposed
in process arguments. This command never creates an identity in the IDP.
`),
		cmds.WithFlags(
			fields.New("db-driver", fields.TypeChoice, fields.WithChoices("postgres", "sqlite"), fields.WithDefault("postgres"), fields.WithHelp("Database dialect")),
			fields.New("db-dsn-file", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Path to a file containing only the database DSN")),
			fields.New("apply-schema", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Apply the current appauth and audit schemas before reconciling")),
			fields.New("issuer", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Absolute HTTPS OIDC issuer URL")),
			fields.New("subject", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Immutable OIDC subject")),
			fields.New("email", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Verified identity email metadata")),
			fields.New("display-name", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Application display name metadata")),
			fields.New("organization-id", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Initial application organization and tenant ID")),
			fields.New("organization-slug", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Initial application organization slug")),
			fields.New("organization-name", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Initial application organization name")),
			fields.New("operator-id", fields.TypeString, fields.WithDefault("deployment-operator"), fields.WithHelp("Non-secret operator identifier written to audit")),
		),
	)}
	return &providerapi.CommandSet{Commands: []cmds.Command{command}}, nil
}

func (c *bootstrapAdminCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	settings := bootstrapAdminSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return errors.Wrap(err, "decode bootstrap-admin settings")
	}
	dsn, err := readDSNFile(settings.DBDSNFile)
	if err != nil {
		return err
	}
	driver, dialect, appDialect, auditDialect, err := bootstrapDialects(settings.DBDriver)
	if err != nil {
		return err
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return errors.Wrap(err, "open administrator bootstrap database")
	}
	defer func() { _ = db.Close() }()
	if err := db.PingContext(ctx); err != nil {
		return errors.Wrap(err, "connect to administrator bootstrap database")
	}
	if settings.ApplySchema {
		appStore, err := appauthsql.New(appauthsql.Config{DB: db, Dialect: appDialect})
		if err != nil {
			return err
		}
		if err := appStore.ApplySchema(ctx); err != nil {
			return err
		}
		auditStore, err := auditsql.New(auditsql.Config{DB: db, Dialect: auditDialect})
		if err != nil {
			return err
		}
		if err := auditStore.ApplySchema(ctx); err != nil {
			return err
		}
	}
	reconciler, err := adminbootstrap.New(db, dialect, nil)
	if err != nil {
		return err
	}
	result, err := reconciler.BootstrapAdmin(ctx, adminbootstrap.Request{
		Issuer: settings.Issuer, Subject: settings.Subject, Email: settings.Email, DisplayName: settings.DisplayName,
		OrganizationID: settings.OrganizationID, OrganizationSlug: settings.OrganizationSlug,
		OrganizationName: settings.OrganizationName, OperatorID: settings.OperatorID,
	})
	if err != nil {
		return err
	}
	return gp.AddRow(ctx, types.NewRow(
		types.MRP("user_id", result.UserID),
		types.MRP("organization_id", result.OrganizationID),
		types.MRP("role", result.Role),
		types.MRP("status", "reconciled"),
	))
}

func readDSNFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("database DSN file is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", errors.Wrap(err, "stat database DSN file")
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxDSNFileBytes {
		return "", errors.Errorf("database DSN file must be a non-empty regular file no larger than %d bytes", maxDSNFileBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", errors.Wrap(err, "read database DSN file")
	}
	dsn := strings.TrimSpace(string(data))
	if dsn == "" || strings.ContainsAny(dsn, "\r\n") {
		return "", errors.New("database DSN file must contain exactly one non-empty line")
	}
	return dsn, nil
}

func bootstrapDialects(value string) (string, adminbootstrap.Dialect, appauthsql.Dialect, auditsql.Dialect, error) {
	switch strings.TrimSpace(value) {
	case "postgres":
		return "postgres", adminbootstrap.DialectPostgres, appauthsql.DialectPostgres, auditsql.DialectPostgres, nil
	case "sqlite":
		return "sqlite3", adminbootstrap.DialectSQLite, appauthsql.DialectSQLite, auditsql.DialectSQLite, nil
	default:
		return "", "", "", "", fmt.Errorf("unsupported bootstrap database driver %q", value)
	}
}

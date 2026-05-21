package provision

import (
	"strings"
	"testing"
)

func TestValidateRequestAcceptsSafeInput(t *testing.T) {
	t.Parallel()

	errs := ValidateRequest(Request{
		RoleName:     "app_user",
		DatabaseName: "app_db",
		RolePassword: "long-enough-password",
	})
	if len(errs) != 0 {
		t.Fatalf("ValidateRequest() errors = %#v", errs)
	}
}

func TestValidateRequestRejectsUnsafeInput(t *testing.T) {
	t.Parallel()

	errs := ValidateRequest(Request{
		RoleName:     "1bad",
		DatabaseName: "postgres",
		RolePassword: "short",
	})
	if len(errs) < 3 {
		t.Fatalf("ValidateRequest() errors = %#v, want role, database and password errors", errs)
	}
}

func TestQuoteHelpersEscapeSQL(t *testing.T) {
	t.Parallel()

	if got := QuoteIdentifier(`app"user`); got != `"app""user"` {
		t.Fatalf("QuoteIdentifier() = %q", got)
	}
	if got := QuoteLiteral(`pa'ss`); got != `'pa''ss'` {
		t.Fatalf("QuoteLiteral() = %q", got)
	}
}

func TestPreviewSQLContainsExpectedStatements(t *testing.T) {
	t.Parallel()

	sql := PreviewSQL(Request{
		RoleName:     "app_user",
		DatabaseName: "app_db",
		RolePassword: "pa'ssword-123",
	})

	for _, want := range []string{
		`CREATE USER "app_user" WITH PASSWORD 'pa''ssword-123';`,
		`CREATE DATABASE "app_db" OWNER "app_user";`,
		`\c "app_db"`,
		`ALTER DEFAULT PRIVILEGES IN SCHEMA public`,
		`\c postgres`,
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("PreviewSQL() missing %q in:\n%s", want, sql)
		}
	}
}

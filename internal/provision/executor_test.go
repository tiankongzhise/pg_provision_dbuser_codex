package provision

import (
	"context"
	"errors"
	"strings"
	"testing"

	"pg_provision_dbuser_codex/internal/config"
)

type fakeRunner struct {
	roleExists     bool
	databaseExists bool
	failAdminAt    string
	adminSQL       []string
	databaseSQL    []string
	closed         bool
}

func (f *fakeRunner) RoleExists(context.Context, string) (bool, error) {
	return f.roleExists, nil
}

func (f *fakeRunner) DatabaseExists(context.Context, string) (bool, error) {
	return f.databaseExists, nil
}

func (f *fakeRunner) ExecAdmin(_ context.Context, sql string) error {
	f.adminSQL = append(f.adminSQL, sql)
	if f.failAdminAt != "" && strings.Contains(sql, f.failAdminAt) {
		return errors.New("admin failed")
	}
	return nil
}

func (f *fakeRunner) ExecDatabase(_ context.Context, databaseName string, sql string) error {
	f.databaseSQL = append(f.databaseSQL, databaseName+":"+sql)
	return nil
}

func (f *fakeRunner) Close() {
	f.closed = true
}

func TestExecutorRunsProvisionSteps(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	executor := NewExecutor(config.PostgresConfig{}, func(context.Context, config.PostgresConfig) (Runner, error) {
		return runner, nil
	})

	result := executor.Execute(context.Background(), Request{
		RoleName:     "app_user",
		DatabaseName: "app_db",
		RolePassword: "long-enough-password",
	})

	if !result.Success {
		t.Fatalf("Execute() success = false, steps = %#v", result.Steps)
	}
	if len(result.Steps) != 10 {
		t.Fatalf("len(steps) = %d, want 10", len(result.Steps))
	}
	if len(runner.adminSQL) != 3 {
		t.Fatalf("admin SQL count = %d, want create user/create database/grant connect", len(runner.adminSQL))
	}
	if len(runner.databaseSQL) != 5 {
		t.Fatalf("database SQL count = %d, want schema/table/sequence/default privileges", len(runner.databaseSQL))
	}
	if !runner.closed {
		t.Fatal("runner was not closed")
	}
	if strings.Contains(result.Steps[2].DisplaySQL, "long-enough-password") {
		t.Fatalf("display SQL leaked password: %q", result.Steps[2].DisplaySQL)
	}
	if !strings.Contains(result.Steps[2].SQL, "long-enough-password") {
		t.Fatalf("execution SQL missing real password: %q", result.Steps[2].SQL)
	}
}

func TestExecutorStopsWhenRoleExists(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{roleExists: true}
	executor := NewExecutor(config.PostgresConfig{}, func(context.Context, config.PostgresConfig) (Runner, error) {
		return runner, nil
	})

	result := executor.Execute(context.Background(), Request{
		RoleName:     "app_user",
		DatabaseName: "app_db",
		RolePassword: "long-enough-password",
	})

	if result.Success {
		t.Fatal("Execute() success = true, want false")
	}
	if len(result.Steps) != 1 || result.Steps[0].Status != StepFailed {
		t.Fatalf("steps = %#v, want one failed existence step", result.Steps)
	}
	if len(runner.adminSQL) != 0 || len(runner.databaseSQL) != 0 {
		t.Fatalf("SQL executed despite existing role: admin=%#v database=%#v", runner.adminSQL, runner.databaseSQL)
	}
}

func TestExecutorStopsWhenDatabaseExists(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{databaseExists: true}
	executor := NewExecutor(config.PostgresConfig{}, func(context.Context, config.PostgresConfig) (Runner, error) {
		return runner, nil
	})

	result := executor.Execute(context.Background(), Request{
		RoleName:     "app_user",
		DatabaseName: "app_db",
		RolePassword: "long-enough-password",
	})

	if result.Success {
		t.Fatal("Execute() success = true, want false")
	}
	if len(result.Steps) != 2 || result.Steps[1].Status != StepFailed {
		t.Fatalf("steps = %#v, want database existence failure after role check", result.Steps)
	}
	if len(runner.adminSQL) != 0 || len(runner.databaseSQL) != 0 {
		t.Fatalf("SQL executed despite existing database: admin=%#v database=%#v", runner.adminSQL, runner.databaseSQL)
	}
}

func TestExecutorRedactsPasswordOnCreateUserFailure(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{failAdminAt: "CREATE USER"}
	executor := NewExecutor(config.PostgresConfig{}, func(context.Context, config.PostgresConfig) (Runner, error) {
		return runner, nil
	})

	result := executor.Execute(context.Background(), Request{
		RoleName:     "app_user",
		DatabaseName: "app_db",
		RolePassword: "long-enough-password",
	})

	if result.Success {
		t.Fatal("Execute() success = true, want false")
	}
	failed := result.Steps[len(result.Steps)-1]
	if !strings.Contains(failed.SQL, "long-enough-password") {
		t.Fatalf("raw SQL missing password: %q", failed.SQL)
	}
	if strings.Contains(failed.DisplaySQL, "long-enough-password") {
		t.Fatalf("display SQL leaked password: %q", failed.DisplaySQL)
	}
}

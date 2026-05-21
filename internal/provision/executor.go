package provision

import (
	"context"
	"errors"
	"fmt"

	"pg_provision_dbuser_codex/internal/config"
)

type StepStatus string

const (
	StepPending StepStatus = "pending"
	StepSuccess StepStatus = "success"
	StepFailed  StepStatus = "failed"
)

type StepResult struct {
	Name   string
	SQL    string
	Status StepStatus
	Error  string
}

type Result struct {
	RoleName     string
	DatabaseName string
	Success      bool
	Steps        []StepResult
}

type Runner interface {
	RoleExists(ctx context.Context, roleName string) (bool, error)
	DatabaseExists(ctx context.Context, databaseName string) (bool, error)
	ExecAdmin(ctx context.Context, sql string) error
	ExecDatabase(ctx context.Context, databaseName string, sql string) error
	Close()
}

type RunnerFactory func(ctx context.Context, cfg config.PostgresConfig) (Runner, error)

type Executor struct {
	cfg        config.PostgresConfig
	newRunner  RunnerFactory
	stepPlanFn func(Request) []plannedStep
}

type plannedStep struct {
	Name     string
	SQL      string
	Database string
}

func NewExecutor(cfg config.PostgresConfig, factory RunnerFactory) *Executor {
	return &Executor{
		cfg:        cfg,
		newRunner:  factory,
		stepPlanFn: buildPlan,
	}
}

func (e *Executor) Execute(ctx context.Context, req Request) Result {
	req = req.Normalized()
	result := Result{
		RoleName:     req.RoleName,
		DatabaseName: req.DatabaseName,
	}

	runner, err := e.newRunner(ctx, e.cfg)
	if err != nil {
		result.Steps = append(result.Steps, StepResult{
			Name:   "连接 PostgreSQL 管理库",
			Status: StepFailed,
			Error:  err.Error(),
		})
		return result
	}
	defer runner.Close()

	if exists, err := runner.RoleExists(ctx, req.RoleName); err != nil {
		result.failCheck("检查目标用户是否存在", err)
		return result
	} else if exists {
		result.failCheck("检查目标用户是否存在", fmt.Errorf("用户 %s 已存在，已停止执行", req.RoleName))
		return result
	}
	result.addSuccess("检查目标用户是否存在", "")

	if exists, err := runner.DatabaseExists(ctx, req.DatabaseName); err != nil {
		result.failCheck("检查目标数据库是否存在", err)
		return result
	} else if exists {
		result.failCheck("检查目标数据库是否存在", fmt.Errorf("数据库 %s 已存在，已停止执行", req.DatabaseName))
		return result
	}
	result.addSuccess("检查目标数据库是否存在", "")

	for _, step := range e.stepPlanFn(req) {
		err := runner.ExecAdmin(ctx, step.SQL)
		if step.Database != "" {
			err = runner.ExecDatabase(ctx, step.Database, step.SQL)
		}
		if err != nil {
			result.addFailed(step.Name, step.SQL, err)
			return result
		}
		result.addSuccess(step.Name, step.SQL)
	}

	result.Success = true
	return result
}

func (r *Result) failCheck(name string, err error) {
	if err == nil {
		err = errors.New("未知错误")
	}
	r.Steps = append(r.Steps, StepResult{
		Name:   name,
		Status: StepFailed,
		Error:  err.Error(),
	})
}

func (r *Result) addSuccess(name, sql string) {
	r.Steps = append(r.Steps, StepResult{
		Name:   name,
		SQL:    sql,
		Status: StepSuccess,
	})
}

func (r *Result) addFailed(name, sql string, err error) {
	message := "未知错误"
	if err != nil {
		message = err.Error()
	}
	r.Steps = append(r.Steps, StepResult{
		Name:   name,
		SQL:    sql,
		Status: StepFailed,
		Error:  message,
	})
}

func buildPlan(req Request) []plannedStep {
	req = req.Normalized()
	role := QuoteIdentifier(req.RoleName)
	database := QuoteIdentifier(req.DatabaseName)
	password := QuoteLiteral(req.RolePassword)

	return []plannedStep{
		{
			Name: "创建专属登录用户",
			SQL:  fmt.Sprintf("CREATE USER %s WITH PASSWORD %s", role, password),
		},
		{
			Name: "创建专属数据库",
			SQL:  fmt.Sprintf("CREATE DATABASE %s OWNER %s", database, role),
		},
		{
			Name: "授予数据库连接权限",
			SQL:  fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s", database, role),
		},
		{
			Name:     "授予 public schema 建表权限",
			SQL:      fmt.Sprintf("GRANT CREATE ON SCHEMA public TO %s", role),
			Database: req.DatabaseName,
		},
		{
			Name:     "授予当前所有表权限",
			SQL:      fmt.Sprintf("GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO %s", role),
			Database: req.DatabaseName,
		},
		{
			Name:     "授予当前所有序列权限",
			SQL:      fmt.Sprintf("GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO %s", role),
			Database: req.DatabaseName,
		},
		{
			Name:     "设置未来新表默认权限",
			SQL:      fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %s", role),
			Database: req.DatabaseName,
		},
		{
			Name:     "设置未来新序列默认权限",
			SQL:      fmt.Sprintf("ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO %s", role),
			Database: req.DatabaseName,
		},
	}
}

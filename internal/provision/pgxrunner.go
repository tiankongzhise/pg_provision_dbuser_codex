package provision

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5"

	"pg_provision_dbuser_codex/internal/config"
)

type pgxRunner struct {
	cfg      config.PostgresConfig
	admin    *pgx.Conn
	database map[string]*pgx.Conn
}

func NewPGXRunner(ctx context.Context, cfg config.PostgresConfig) (Runner, error) {
	admin, err := pgx.Connect(ctx, connString(cfg, cfg.AdminDB))
	if err != nil {
		return nil, fmt.Errorf("连接管理库失败: %w", err)
	}
	return &pgxRunner{
		cfg:      cfg,
		admin:    admin,
		database: make(map[string]*pgx.Conn),
	}, nil
}

func (r *pgxRunner) RoleExists(ctx context.Context, roleName string) (bool, error) {
	var exists bool
	err := r.admin.QueryRow(ctx, "select exists(select 1 from pg_roles where rolname = $1)", roleName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("检查用户是否存在失败: %w", err)
	}
	return exists, nil
}

func (r *pgxRunner) DatabaseExists(ctx context.Context, databaseName string) (bool, error) {
	var exists bool
	err := r.admin.QueryRow(ctx, "select exists(select 1 from pg_database where datname = $1)", databaseName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("检查数据库是否存在失败: %w", err)
	}
	return exists, nil
}

func (r *pgxRunner) ExecAdmin(ctx context.Context, sql string) error {
	if _, err := r.admin.Exec(ctx, sql); err != nil {
		return fmt.Errorf("管理库执行失败: %w", err)
	}
	return nil
}

func (r *pgxRunner) ExecDatabase(ctx context.Context, databaseName string, sql string) error {
	conn, err := r.databaseConn(ctx, databaseName)
	if err != nil {
		return err
	}
	if _, err := conn.Exec(ctx, sql); err != nil {
		return fmt.Errorf("业务库执行失败: %w", err)
	}
	return nil
}

func (r *pgxRunner) Close() {
	for _, conn := range r.database {
		_ = conn.Close(context.Background())
	}
	if r.admin != nil {
		_ = r.admin.Close(context.Background())
	}
}

func (r *pgxRunner) databaseConn(ctx context.Context, databaseName string) (*pgx.Conn, error) {
	if conn, ok := r.database[databaseName]; ok {
		return conn, nil
	}
	conn, err := pgx.Connect(ctx, connString(r.cfg, databaseName))
	if err != nil {
		return nil, fmt.Errorf("连接业务库失败: %w", err)
	}
	r.database[databaseName] = conn
	return conn, nil
}

func connString(cfg config.PostgresConfig, database string) string {
	values := url.Values{}
	values.Set("sslmode", cfg.SSLMode)
	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.SuperUser, cfg.SuperPassword),
		Host:     cfg.Host + ":" + cfg.Port,
		Path:     database,
		RawQuery: values.Encode(),
	}).String()
}

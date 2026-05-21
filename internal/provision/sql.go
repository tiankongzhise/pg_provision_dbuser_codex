package provision

import (
	"fmt"
	"strings"
)

func PreviewSQL(req Request) string {
	req = req.Normalized()
	role := QuoteIdentifier(req.RoleName)
	database := QuoteIdentifier(req.DatabaseName)
	password := QuoteLiteral(req.RolePassword)

	var b strings.Builder
	fmt.Fprintf(&b, "-- 1. 创建专属登录用户\n")
	fmt.Fprintf(&b, "CREATE USER %s WITH PASSWORD %s;\n\n", role, password)
	fmt.Fprintf(&b, "-- 2. 创建专属数据库，并指定该用户为所有者\n")
	fmt.Fprintf(&b, "CREATE DATABASE %s OWNER %s;\n\n", database, role)
	fmt.Fprintf(&b, "-- 3. 赋予该用户连接自己数据库的权限\n")
	fmt.Fprintf(&b, "GRANT CONNECT ON DATABASE %s TO %s;\n\n", database, role)
	fmt.Fprintf(&b, "-- 4. 切换到该专属数据库内部\n")
	fmt.Fprintf(&b, "\\c %s\n\n", database)
	fmt.Fprintf(&b, "-- 5. 赋予该用户在自己库的 public 模式下建表、操作对象的完整权限\n")
	fmt.Fprintf(&b, "GRANT CREATE ON SCHEMA public TO %s;\n\n", role)
	fmt.Fprintf(&b, "-- 6. 授予对当前所有表的增删改查权限\n")
	fmt.Fprintf(&b, "GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO %s;\n\n", role)
	fmt.Fprintf(&b, "-- 7. 授予对当前所有序列的权限\n")
	fmt.Fprintf(&b, "GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO %s;\n\n", role)
	fmt.Fprintf(&b, "-- 8. 设置未来新建的表，自动赋予该用户增删改查权限\n")
	fmt.Fprintf(&b, "ALTER DEFAULT PRIVILEGES IN SCHEMA public\n")
	fmt.Fprintf(&b, "GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %s;\n\n", role)
	fmt.Fprintf(&b, "-- 9. 设置未来新建的序列，自动赋予该用户使用权限\n")
	fmt.Fprintf(&b, "ALTER DEFAULT PRIVILEGES IN SCHEMA public\n")
	fmt.Fprintf(&b, "GRANT USAGE, SELECT ON SEQUENCES TO %s;\n\n", role)
	fmt.Fprintf(&b, "-- 10. 操作完成，切回超级管理员的 postgres 数据库\n")
	fmt.Fprintf(&b, "\\c postgres\n")
	return b.String()
}

func QuoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func QuoteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

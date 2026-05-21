# PostgreSQL 用户开通工具开发文档

## 1. 技术方案

项目使用 Go 开发本地 Web 管理工具。HTTP 层使用标准库 `net/http`，页面渲染使用 `html/template`，PostgreSQL 驱动使用 `github.com/jackc/pgx/v5`。

第一版只支持单个 PostgreSQL 目标环境，所有应用登录配置和数据库超级账号配置均从 `.env` 读取。

## 2. 项目结构

```text
cmd/pg-provision/main.go          程序入口
internal/app/                     HTTP 路由、模板和页面处理
internal/config/                  .env 加载和配置校验
internal/password/                强密码生成
internal/provision/               SQL 预览、输入校验和执行编排
internal/session/                 Cookie 会话签名和校验
web/templates/                    HTML 模板
web/static/                       CSS 和浏览器脚本
docs/                             产品和开发文档
```

## 3. 配置项

`.env` 由程序启动时读取。缺少必要字段时，程序直接启动失败并输出字段名。

```env
APP_ADDR=127.0.0.1:8080
APP_LOGIN_USER=admin
APP_LOGIN_KEY=change-me
APP_SESSION_SECRET=change-me-long-random-secret

PG_HOST=127.0.0.1
PG_PORT=5432
PG_ADMIN_DB=postgres
PG_SUPERUSER=postgres
PG_SUPER_PASSWORD=postgres
PG_SSLMODE=disable
```

配置解析规则：

- 支持空行和 `#` 注释。
- 支持 `KEY=value`。
- 支持值两侧双引号或单引号。
- `APP_ADDR` 为空时默认 `127.0.0.1:8080`。
- `PG_PORT` 为空时默认 `5432`。
- `PG_ADMIN_DB` 为空时默认 `postgres`。
- `PG_SSLMODE` 为空时默认 `disable`。

## 4. HTTP 路由

- `GET /login`：展示登录页。
- `POST /login`：校验 `APP_LOGIN_USER` 和 `APP_LOGIN_KEY`，成功后写入签名 Cookie。
- `POST /logout`：清除会话 Cookie。
- `GET /`：展示业务用户开通表单，需要登录。
- `GET /api/password`：返回 JSON 强密码，需要登录。
- `POST /preview`：校验表单并生成 SQL 预览，需要登录。
- `POST /execute`：根据服务端草稿执行 PostgreSQL 操作，需要登录。

所有业务页面必须经过认证中间件。未登录访问业务页面时跳转 `/login`。

## 5. 会话设计

会话 Cookie 名称为 `pg_provision_session`。Cookie 值由用户名、过期时间和 HMAC-SHA256 签名组成，签名密钥为 `APP_SESSION_SECRET`。

会话默认有效期 8 小时。退出登录时写入过期 Cookie。第一版不维护服务端用户表，不支持多账号和权限分级。

## 6. 输入校验

业务用户名和数据库名使用统一规则：

- 正则：`^[A-Za-z_][A-Za-z0-9_]{0,62}$`
- 数据库名禁止使用 `postgres`、`template0`、`template1`

业务密码规则：

- 最少 12 字符
- 不在服务端日志输出
- 预览草稿仅保存在服务端内存中，不写入隐藏字段

## 7. SQL 预览

预览 SQL 使用安全引用后的标识符和字面量生成。标识符双引号转义，字符串单引号转义。

预览内容保持接近人工模板：

```sql
CREATE USER "app_user" WITH PASSWORD '...';
CREATE DATABASE "app_db" OWNER "app_user";
GRANT CONNECT ON DATABASE "app_db" TO "app_user";
\c "app_db"
GRANT CREATE ON SCHEMA public TO "app_user";
...
\c postgres
```

预览草稿使用随机 ID 存放在内存中，绑定当前登录用户，默认 15 分钟过期。执行完成或过期后不可复用。

## 8. PostgreSQL 执行编排

执行服务通过接口抽象数据库操作，便于单元测试。

步骤固定为：

1. 连接 `PG_ADMIN_DB`。
2. 查询 `pg_roles`，目标用户存在则停止。
3. 查询 `pg_database`，目标数据库存在则停止。
4. 执行 `CREATE USER`。
5. 执行 `CREATE DATABASE`。
6. 执行 `GRANT CONNECT ON DATABASE`。
7. 连接目标业务数据库。
8. 执行 `GRANT CREATE ON SCHEMA public`。
9. 执行 `GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public`。
10. 执行 `GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public`。
11. 执行 `ALTER DEFAULT PRIVILEGES ... ON TABLES`。
12. 执行 `ALTER DEFAULT PRIVILEGES ... ON SEQUENCES`。

`CREATE DATABASE` 不能放入事务。中途失败时返回已完成步骤、失败步骤和错误原因，不自动回滚。

## 9. 页面实现

页面保持简洁的管理工具风格：

- 登录页只展示账号和密钥输入。
- 首页表单展示当前 PostgreSQL 目标摘要，不展示超级账号密码。
- 预览页展示 SQL 代码块和确认执行按钮。
- 结果页使用步骤列表展示成功、失败和未执行状态。

静态文件通过 Go 内嵌 `embed` 打包，方便单文件运行。

## 10. 测试策略

单元测试覆盖：

- `.env` 解析、默认值和缺失配置错误。
- 登录成功、登录失败、未登录重定向。
- 用户名、数据库名和密码校验。
- SQL 标识符和字符串转义。
- SQL 预览内容。
- 用户已存在、数据库已存在时停止。
- 执行步骤编排和失败结果。

真实 PostgreSQL 连接作为手工验收，不作为默认测试依赖。

## 11. 提交策略

每个功能点单独提交。提交信息使用中文，正文包含：

- 实现内容
- 关键细节
- 验证方式

提交顺序与产品计划保持一致：产品文档、开发文档、项目骨架、配置登录、表单密码、预览草稿、PostgreSQL 执行、测试补充。

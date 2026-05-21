# pg_provision_dbuser_codex

一个本地 Go Web 工具，用于自动化开通 PostgreSQL 业务用户、专属数据库和常用权限。

## 功能

- 使用 `.env` 中的管理员账号和密钥登录。
- 从 `.env` 读取 PostgreSQL 超级账号连接信息。
- 通过本地 Web 页面填写业务用户名、数据库名和业务用户密码。
- 支持一键生成强密码。
- 先预览 SQL，再确认执行。
- 用户名或数据库名已存在时停止执行，不跳过、不覆盖。
- 执行结果按步骤展示；中途失败不会自动删除已经创建的对象。

## 快速开始

复制配置模板：

```powershell
Copy-Item .env.example .env
```

编辑 `.env`：

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

启动服务：

```powershell
go run ./cmd/pg-provision
```

浏览器访问：

```text
http://127.0.0.1:8080
```

## 验证

运行默认测试：

```powershell
go test ./...
```

可选真实数据库验收：

1. 使用测试 PostgreSQL 超级账号配置 `.env`。
2. 启动服务并登录。
3. 创建一个全新的业务用户名和数据库名。
4. 使用新用户连接新数据库，验证可以连接、建表、插入数据并使用自增序列。
5. 使用相同用户名或数据库名再次提交，确认工具会停止并提示已存在。

## 文档

- 产品文档：[docs/product.md](docs/product.md)
- 开发文档：[docs/development.md](docs/development.md)

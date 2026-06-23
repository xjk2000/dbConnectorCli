# dbConnector CLI 设计文档

## 目标

`dbConnector` 是一个面向 AI Agent 调用的本机数据库命令行工具。它不是完整的交互式数据库客户端，而是一个受控执行器，提供稳定、可审计、机器可解析的数据库访问能力。

第一版目标：

- 使用 Go 实现，尽量减少依赖和运行时内存占用。
- 以单个 CLI 二进制运行在本机。
- 允许 AI 在沙箱或本机环境中通过命令行调用。
- 支持 MySQL 和 Redis。
- 配置文件放在用户全局目录。
- 默认禁止写操作，写操作必须通过参数显式开启，并受全局配置和 profile 配置共同约束。
- 所有输出默认使用 JSON，方便 AI 解析。

## 非目标

第一版暂不实现：

- 交互式数据库 Shell。
- SQL 自动生成或自动修复。
- migration 管理。
- 备份、恢复、导入、导出。
- Redis Pub/Sub、Stream、Cluster 管理。
- Redis Lua 脚本执行。
- 长事务会话。
- Web UI。

## 命令总览

建议可执行文件名：

```bash
dbc
```

也可以在发布时提供 `dbconnector` 作为长命令别名。

### 通用命令

```bash
dbc config path
dbc profile list
dbc profile test --profile local-mysql
```

### MySQL 命令

```bash
dbc mysql databases --profile local-mysql
dbc mysql tables --profile local-mysql --database app
dbc mysql table --profile local-mysql --database app --table users
dbc mysql query --profile local-mysql --sql "select * from users limit 10"
dbc mysql exec --profile local-mysql --sql "update users set name=? where id=?" --params '["Tom",1]' --write
dbc mysql explain --profile local-mysql --sql "select * from users where id = 1"
```

### Redis 命令

```bash
dbc redis ping --profile local-redis
dbc redis info --profile local-redis
dbc redis scan --profile local-redis --pattern "user:*" --limit 50
dbc redis get --profile local-redis --key "user:1"
dbc redis hgetall --profile local-redis --key "user:1"
dbc redis ttl --profile local-redis --key "user:1"
dbc redis type --profile local-redis --key "user:1"
dbc redis set --profile local-redis --key "user:1" --value '{"name":"Tom"}' --write
dbc redis del --profile local-redis --key "user:1" --write
```

## 配置文件

默认配置路径：

```text
~/.dbconnector/config.json
```

可以通过环境变量覆盖：

```bash
DBCONNECTOR_CONFIG=/path/to/config.json dbc profile list
```

### 配置示例

```json
{
  "defaults": {
    "output": "json",
    "timeoutMs": 5000,
    "maxRows": 100,
    "allowWrite": false
  },
  "profiles": [
    {
      "name": "local-mysql",
      "type": "mysql",
      "dsnEnv": "LOCAL_MYSQL_DSN",
      "readonly": false,
      "maxRows": 200,
      "timeoutMs": 8000
    },
    {
      "name": "local-redis",
      "type": "redis",
      "addr": "127.0.0.1:6379",
      "usernameEnv": "LOCAL_REDIS_USERNAME",
      "passwordEnv": "LOCAL_REDIS_PASSWORD",
      "db": 0,
      "readonly": false,
      "timeoutMs": 3000
    }
  ]
}
```

### 配置字段

`defaults`：

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `output` | string | `json` | 输出格式。第一版只支持 `json`。 |
| `timeoutMs` | number | `5000` | 默认命令超时时间。 |
| `maxRows` | number | `100` | MySQL 查询默认最大返回行数，Redis scan 默认最大返回 key 数。 |
| `allowWrite` | boolean | `false` | 全局是否允许写操作。即使为 true，命令仍必须传 `--write`。 |

MySQL profile：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `name` | string | 是 | profile 名称。 |
| `type` | string | 是 | 固定为 `mysql`。 |
| `dsn` | string | 否 | 直接配置 MySQL DSN。不建议用于共享配置。 |
| `dsnEnv` | string | 否 | 从环境变量读取 MySQL DSN。`dsn` 和 `dsnEnv` 至少配置一个。 |
| `readonly` | boolean | 否 | 是否强制只读。 |
| `maxRows` | number | 否 | 覆盖默认最大行数。 |
| `timeoutMs` | number | 否 | 覆盖默认超时时间。 |

Redis profile：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `name` | string | 是 | profile 名称。 |
| `type` | string | 是 | 固定为 `redis`。 |
| `addr` | string | 是 | Redis 地址，例如 `127.0.0.1:6379`。 |
| `usernameEnv` | string | 否 | 从环境变量读取 Redis 用户名。 |
| `passwordEnv` | string | 否 | 从环境变量读取 Redis 密码。 |
| `db` | number | 否 | Redis DB 编号，默认 `0`。 |
| `readonly` | boolean | 否 | 是否强制只读。 |
| `timeoutMs` | number | 否 | 覆盖默认超时时间。 |

## 写操作权限模型

写操作必须同时满足以下条件：

1. 命令传入 `--write`。
2. 全局配置 `defaults.allowWrite` 为 `true`。
3. 当前 profile 的 `readonly` 不为 `true`。
4. 命令本身在允许的写操作白名单内。

如果任意条件不满足，返回 `WRITE_NOT_ALLOWED`。

这样设计的含义是：

- `defaults.allowWrite` 表示这个本机环境整体允许写。
- `profile.readonly` 表示单个连接是否强制只读。
- `--write` 表示本次调用明确请求写入。

## MySQL 安全策略

### 只读命令

`mysql query` 默认只允许以下 SQL 类型：

- `SELECT`
- `SHOW`
- `DESCRIBE`
- `DESC`
- `EXPLAIN`

### 写命令

`mysql exec` 必须传 `--write`，并通过写操作权限模型。

允许的写 SQL 类型：

- `INSERT`
- `UPDATE`
- `DELETE`
- `REPLACE`
- `CREATE`
- `ALTER`
- `DROP`
- `TRUNCATE`

第一版只做轻量级 SQL 类型识别：

- 去除开头空白。
- 跳过开头的 SQL 注释。
- 读取第一个关键字并转为大写。
- 多语句默认拒绝，避免 `SELECT 1; DROP TABLE users` 这种绕过。

### 返回行数限制

`mysql query` 必须限制返回行数：

- 如果 SQL 已包含 `LIMIT`，按数据库执行结果返回，但仍可在客户端截断到 `maxRows`。
- 如果 SQL 不包含 `LIMIT`，第一版优先在客户端最多读取 `maxRows + 1` 行，并在超过时设置 `truncated: true`。

后续版本可以考虑更可靠的 SQL AST 改写。

## Redis 安全策略

### 只读命令白名单

第一版只暴露明确的子命令，不提供任意 Redis 命令透传。

允许的只读能力：

- `PING`
- `INFO`
- `SCAN`
- `GET`
- `MGET`
- `HGET`
- `HGETALL`
- `TTL`
- `TYPE`
- `EXISTS`

### 写命令白名单

写命令必须传 `--write`，并通过写操作权限模型。

第一版允许：

- `SET`
- `DEL`
- `HSET`
- `EXPIRE`

### 禁止命令

第一版不暴露以下危险能力：

- `KEYS`
- `FLUSHALL`
- `FLUSHDB`
- `CONFIG`
- `SHUTDOWN`
- `EVAL`
- `EVALSHA`
- `SCRIPT`
- `MIGRATE`
- `RESTORE`

`redis scan` 使用 Redis `SCAN` 实现，不调用 `KEYS`。

## 参数规范

通用参数：

| 参数 | 说明 |
| --- | --- |
| `--profile` | 指定 profile 名称。 |
| `--timeout-ms` | 覆盖本次命令超时时间。 |
| `--limit` | 覆盖本次命令最大返回数量。 |
| `--write` | 显式请求写操作。 |

MySQL 参数：

| 参数 | 说明 |
| --- | --- |
| `--database` | 数据库名。 |
| `--table` | 表名。 |
| `--sql` | SQL 文本。 |
| `--params` | JSON 数组格式的 SQL 参数。 |

Redis 参数：

| 参数 | 说明 |
| --- | --- |
| `--key` | Redis key。 |
| `--pattern` | scan pattern。 |
| `--value` | 写入值。 |
| `--field` | hash field。 |
| `--ttl` | 过期时间，单位秒。 |

## 输出协议

所有命令默认输出 JSON。成功时：

```json
{
  "ok": true,
  "engine": "mysql",
  "profile": "local-mysql",
  "type": "query_result",
  "columns": [
    { "name": "id", "type": "BIGINT" },
    { "name": "name", "type": "VARCHAR" }
  ],
  "rows": [
    { "id": 1, "name": "Tom" }
  ],
  "rowCount": 1,
  "truncated": false,
  "elapsedMs": 12
}
```

失败时：

```json
{
  "ok": false,
  "engine": "mysql",
  "profile": "local-mysql",
  "error": {
    "code": "WRITE_NOT_ALLOWED",
    "message": "write operation requires --write, global allowWrite, and non-readonly profile",
    "retryable": false
  },
  "elapsedMs": 1
}
```

### 通用响应字段

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `ok` | boolean | 命令是否成功。 |
| `engine` | string | `mysql`、`redis` 或 `system`。 |
| `profile` | string | 当前 profile 名称。无 profile 命令可为空。 |
| `type` | string | 结果类型。 |
| `elapsedMs` | number | 命令耗时。 |
| `error` | object | 失败时的标准错误对象。 |

### 错误对象

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `code` | string | 稳定错误码。 |
| `message` | string | 给人和 AI 阅读的错误说明。 |
| `retryable` | boolean | 是否适合重试。 |
| `driverCode` | string | 可选，数据库驱动原始错误码。 |

## 错误码

第一版错误码：

| 错误码 | 说明 |
| --- | --- |
| `CONFIG_NOT_FOUND` | 配置文件不存在。 |
| `CONFIG_INVALID` | 配置文件格式错误。 |
| `PROFILE_NOT_FOUND` | profile 不存在。 |
| `PROFILE_TYPE_MISMATCH` | 命令类型和 profile 类型不匹配。 |
| `ENV_NOT_FOUND` | 配置引用的环境变量不存在。 |
| `CONNECT_FAILED` | 数据库连接失败。 |
| `TIMEOUT` | 命令超时。 |
| `WRITE_NOT_ALLOWED` | 写操作未被允许。 |
| `SQL_NOT_ALLOWED` | SQL 类型不在允许范围内。 |
| `MULTI_STATEMENT_NOT_ALLOWED` | 检测到多语句。 |
| `PARAMS_INVALID` | 参数 JSON 格式错误。 |
| `DRIVER_ERROR` | 数据库驱动返回错误。 |
| `INTERNAL_ERROR` | 未分类内部错误。 |

## 审计日志

第一版可以先预留结构，不默认启用。

后续建议支持：

```json
{
  "audit": {
    "enabled": true,
    "path": "~/.dbconnector/audit.log"
  }
}
```

每行 JSONL：

```json
{"time":"2026-06-23T15:00:00+08:00","engine":"mysql","profile":"local-mysql","command":"query","write":false,"ok":true,"elapsedMs":12,"rowCount":10}
```

审计日志不得记录密码、完整 DSN、Redis 密码或其他凭据。

## Go 项目结构

建议结构：

```text
.
├── cmd
│   └── dbc
│       └── main.go
├── internal
│   ├── app
│   │   └── app.go
│   ├── config
│   │   └── config.go
│   ├── output
│   │   └── output.go
│   ├── safety
│   │   ├── mysql.go
│   │   └── redis.go
│   ├── mysql
│   │   ├── client.go
│   │   └── commands.go
│   └── redis
│       ├── client.go
│       └── commands.go
├── docs
│   └── design.md
├── go.mod
└── README.md
```

### 依赖建议

第一版尽量只使用：

- Go 标准库：命令解析、JSON、上下文、时间、文件读写。
- `github.com/go-sql-driver/mysql`
- `github.com/redis/go-redis/v9`

暂不使用 Cobra，直接基于标准库解析子命令，降低依赖和启动成本。

## 实现顺序

建议按以下顺序实现：

1. 初始化 Go module 和 CLI 入口。
2. 实现 JSON 输出和标准错误模型。
3. 实现全局配置读取、默认值、profile 查找。
4. 实现 `config path`、`profile list`。
5. 实现写操作权限判定。
6. 实现 MySQL 连接和 `profile test`。
7. 实现 MySQL schema 查询。
8. 实现 MySQL `query`、`exec`、SQL 安全检查。
9. 实现 Redis 连接和 `profile test`。
10. 实现 Redis 只读命令。
11. 实现 Redis 写命令。
12. 增加单元测试和最小 README。

## MVP 验收标准

第一版完成时应满足：

- 在无配置文件时，命令返回标准 JSON 错误。
- `dbc config path` 能输出当前使用的配置路径。
- `dbc profile list` 能列出 MySQL 和 Redis profile，且不会泄露敏感环境变量值。
- `dbc profile test` 能测试 MySQL 和 Redis 连接。
- MySQL 只读查询默认可用，写 SQL 默认被拒绝。
- MySQL 写操作只有在 `--write`、`defaults.allowWrite=true`、`readonly=false` 同时满足时才执行。
- Redis 只读命令默认可用，写命令默认被拒绝。
- Redis 写操作只有在 `--write`、`defaults.allowWrite=true`、`readonly=false` 同时满足时才执行。
- 所有成功和失败输出都可以被 JSON parser 解析。
- 错误信息中不包含密码、完整 DSN 或 Redis 密码。

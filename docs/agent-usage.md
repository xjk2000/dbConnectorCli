# dbConnector Agent Usage

This document is written for AI agents that need to use `dbConnector` through a command line. It is intentionally plain Markdown with stable command examples and JSON response shapes.

## Tool Summary

`dbc` is a local CLI for controlled database access.

Supported engines:

- MySQL
- Redis

Main properties:

- JSON-first output.
- Safe by default.
- Read operations are allowed by default.
- Write operations are blocked unless explicitly enabled.
- Profiles are loaded from the user's global config.
- Sensitive connection data must not be printed.

## Executable

Use this executable when available:

```bash
./bin/dbc
```

Show help:

```bash
./bin/dbc -help
./bin/dbc --help
```

Help output is JSON. Read `usage` for a grouped human-readable command reference, and `commandTree` for a machine-friendly hierarchy of every supported command.

Show version:

```bash
./bin/dbc -version
```

If `./bin/dbc` does not exist, build it:

```bash
go build -o bin/dbc ./cmd/dbc
```

Build with version metadata:

```bash
go build \
  -ldflags "-X dbconnector/internal/version.Version=v0.1.0 -X dbconnector/internal/version.Commit=$(git rev-parse --short HEAD) -X dbconnector/internal/version.BuiltAt=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o bin/dbc ./cmd/dbc
```

Fallback during development:

```bash
go run ./cmd/dbc <command>
```

## Config

Default config path:

```text
~/.dbconnector/config.json
```

Check the active config path:

```bash
./bin/dbc config path
```

Override config path:

```bash
DBCONNECTOR_CONFIG=/path/to/config.json ./bin/dbc profile list
```

Important:

- Do not expose raw DSN values.
- Do not expose passwords.
- `profile list` is designed to avoid printing MySQL DSN values.
- If a profile contains `dsnConfigured: true`, it means a DSN exists but is intentionally hidden.

## Discovery

Always discover available profiles before running database commands unless the user gave a specific profile:

```bash
./bin/dbc profile list
```

Example response:

```json
{
  "ok": true,
  "engine": "system",
  "profile": "",
  "type": "profile_list",
  "profiles": [
    {
      "name": "cactus-staging-mysql",
      "type": "mysql",
      "dsnConfigured": true,
      "readonly": false,
      "maxRows": 100,
      "timeoutMs": 8000
    },
    {
      "name": "cactus-next-redis",
      "type": "redis",
      "addr": "cactus-next.example.cache.amazonaws.com:6379",
      "db": 0,
      "readonly": false,
      "timeoutMs": 5000
    }
  ]
}
```

Test a profile:

```bash
./bin/dbc profile test --profile <profile-name>
```

## JSON Response Contract

Every command writes one JSON object to stdout.

Success shape:

```json
{
  "ok": true,
  "engine": "mysql",
  "profile": "example-profile",
  "type": "query_result",
  "elapsedMs": 12
}
```

Failure shape:

```json
{
  "ok": false,
  "engine": "mysql",
  "profile": "example-profile",
  "error": {
    "code": "WRITE_NOT_ALLOWED",
    "message": "write operation requires --write",
    "retryable": false
  },
  "elapsedMs": 1
}
```

Agent handling rules:

- Parse stdout as JSON.
- If `ok` is `true`, use the result fields.
- If `ok` is `false`, report `error.code` and a short summary of `error.message`.
- Do not assume process exit code alone is enough; failures still produce valid JSON.
- If JSON parsing fails, report that the tool output was not valid JSON.

## Safety Rules

Default behavior:

- MySQL `query` is read-only.
- Redis read commands are allowed.
- Write commands require `--write`.
- Write commands also require global config `defaults.allowWrite=true`.
- Write commands are blocked if the profile has `readonly: true`.

Never run write operations unless the user explicitly asks for a data-changing action.

Write examples:

- MySQL `exec`
- Redis `set`
- Redis `del`

Do not use write commands for exploration.

## MySQL Commands

Use a MySQL profile:

```bash
./bin/dbc profile test --profile <mysql-profile>
```

List visible databases:

```bash
./bin/dbc mysql databases --profile <mysql-profile>
```

List tables in a database:

```bash
./bin/dbc mysql tables --profile <mysql-profile> --database <database-name> --limit 1000
```

Inspect one table:

```bash
./bin/dbc mysql table --profile <mysql-profile> --database <database-name> --table <table-name>
```

Run a read-only query:

```bash
./bin/dbc mysql query --profile <mysql-profile> --sql "select * from users limit 10"
```

Run a parameterized read-only query:

```bash
./bin/dbc mysql query --profile <mysql-profile> --sql "select * from users where id = ?" --params '[123]'
```

Explain a query:

```bash
./bin/dbc mysql explain --profile <mysql-profile> --sql "select * from users where id = 123"
```

Count rows in a table without returning rows:

```bash
./bin/dbc mysql count --profile <mysql-profile> --database <database-name> --table <table-name>
```

Count rows returned by a read-only query without returning rows:

```bash
./bin/dbc mysql count --profile <mysql-profile> --sql "select * from users where status = ?" --params '["active"]'
```

Run a write command only when explicitly requested:

```bash
./bin/dbc mysql exec --profile <mysql-profile> --sql "update users set name = ? where id = ?" --params '["Alice", 123]' --write
```

MySQL read SQL allowlist:

- `SELECT`
- `SHOW`
- `DESCRIBE`
- `DESC`
- `EXPLAIN`

MySQL write SQL allowlist:

- `INSERT`
- `UPDATE`
- `DELETE`
- `REPLACE`
- `CREATE`
- `ALTER`
- `DROP`
- `TRUNCATE`

Multiple SQL statements are rejected.

## Redis Commands

Use a Redis profile:

```bash
./bin/dbc profile test --profile <redis-profile>
```

Ping:

```bash
./bin/dbc redis ping --profile <redis-profile>
```

Get Redis info:

```bash
./bin/dbc redis info --profile <redis-profile>
```

Scan keys:

```bash
./bin/dbc redis scan --profile <redis-profile> --pattern "prefix*" --limit 1000
```

Count keys without returning key names:

```bash
./bin/dbc redis count --profile <redis-profile> --pattern "prefix*"
```

Override the Redis logical DB for a single command:

```bash
./bin/dbc redis count --profile <redis-profile> --db 1 --pattern "prefix*"
```

Get a string key:

```bash
./bin/dbc redis get --profile <redis-profile> --key "user:1"
```

Get all hash fields:

```bash
./bin/dbc redis hgetall --profile <redis-profile> --key "user:1"
```

Get TTL:

```bash
./bin/dbc redis ttl --profile <redis-profile> --key "user:1"
```

Get Redis type:

```bash
./bin/dbc redis type --profile <redis-profile> --key "user:1"
```

Set a key only when explicitly requested:

```bash
./bin/dbc redis set --profile <redis-profile> --key "user:1" --value '{"name":"Alice"}' --ttl 3600 --write
```

Delete a key only when explicitly requested:

```bash
./bin/dbc redis del --profile <redis-profile> --key "user:1" --write
```

Important:

- Use `scan`, not Redis `KEYS`.
- This CLI does not expose dangerous Redis commands such as `FLUSHALL`, `FLUSHDB`, `CONFIG`, `SHUTDOWN`, `EVAL`, or `SCRIPT`.

## Common Tasks

Count visible MySQL databases:

```bash
./bin/dbc mysql databases --profile <mysql-profile>
```

Read the `count` field from the JSON response.

Count tables in a MySQL database:

```bash
./bin/dbc mysql tables --profile <mysql-profile> --database <database-name> --limit 100000
```

Read the `count` field. If `truncated` is `true`, rerun with a larger `--limit`.

Count rows in a MySQL table:

```bash
./bin/dbc mysql count --profile <mysql-profile> --database <database-name> --table <table-name>
```

Read the `count` field. Prefer this command over `select count(*) ...` because it returns a stable JSON shape.

Count rows produced by a read-only MySQL query:

```bash
./bin/dbc mysql count --profile <mysql-profile> --sql "select * from users where status = ?" --params '["active"]'
```

This wraps the read query as a subquery. If the SQL contains `LIMIT`, the count reflects the limited query result.

Count Redis keys matching a prefix:

```bash
./bin/dbc redis count --profile <redis-profile> --pattern "prefix*"
```

Read the `count` field. This command does not return key names, so it is safer for large keyspaces. If the user wants sample keys, use `redis scan` with a small `--limit`.

Find Redis DBs with data:

```bash
./bin/dbc redis info --profile <redis-profile>
```

Parse the `info` string, section `# Keyspace`. Lines look like:

```text
db0:keys=1619,expires=348,avg_ttl=117253440
```

Only DBs with keys are listed by Redis `INFO keyspace`.

## Error Codes

Known error codes:

- `CONFIG_NOT_FOUND`: config file does not exist.
- `CONFIG_INVALID`: config file or profile is invalid.
- `PROFILE_NOT_FOUND`: named profile does not exist.
- `PROFILE_TYPE_MISMATCH`: command engine does not match profile type.
- `ENV_NOT_FOUND`: required environment variable is missing.
- `CONNECT_FAILED`: database connection failed.
- `TIMEOUT`: command timed out.
- `WRITE_NOT_ALLOWED`: write operation blocked.
- `SQL_NOT_ALLOWED`: SQL keyword is not allowed.
- `MULTI_STATEMENT_NOT_ALLOWED`: multiple SQL statements were detected.
- `PARAMS_INVALID`: `--params` is not a valid JSON array.
- `DRIVER_ERROR`: database driver returned an error.
- `USAGE_ERROR`: command arguments are invalid.

## Reporting Guidelines

When reporting results to a user:

- Give the direct answer first.
- Include the command used only when useful.
- Do not paste large JSON responses unless asked.
- Do not print secrets.
- Mention truncation if `truncated: true`.
- Mention that row counts from MySQL `information_schema.tables.table_rows` are approximate for InnoDB when discussing table row estimates.

## Current Local Profiles

If this repo is used on the current machine, the following profiles may exist:

- `cactus-staging-mysql`: MySQL profile.
- `cactus-next-redis`: Redis profile using DB 0 by default.

Agents should still run `./bin/dbc profile list` instead of assuming these profiles always exist.

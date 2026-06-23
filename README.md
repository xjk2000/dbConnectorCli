# dbConnector

面向 AI Agent 调用的本机数据库 CLI 工具。第一版目标是提供稳定、默认安全、JSON 可解析的 MySQL 和 Redis 操作能力。

## 当前进度

已实现：

- `dbc config path`
- `dbc profile list`
- `dbc profile test --profile <name>`
- 全局配置路径解析
- 标准 JSON 成功和错误输出
- MySQL / Redis 连接测试
- 写操作权限模型
- MySQL：`databases`、`tables`、`table`、`query`、`explain`、`exec`
- Redis：`ping`、`info`、`scan`、`get`、`hgetall`、`ttl`、`type`、`set`、`del`
- 示例配置文件

待实现：

- 更多自动化测试，尤其是带真实 MySQL / Redis 的集成测试
- 审计日志
- 发布脚本和二进制构建配置

## 运行

```bash
go run ./cmd/dbc config path
```

构建本机二进制：

```bash
go build -o bin/dbc ./cmd/dbc
```

使用示例配置：

```bash
DBCONNECTOR_CONFIG=docs/config.example.json go run ./cmd/dbc profile list
```

MySQL 只读查询：

```bash
DBCONNECTOR_CONFIG=docs/config.example.json go run ./cmd/dbc mysql query --profile local-mysql --sql "select 1"
```

Redis 读取：

```bash
DBCONNECTOR_CONFIG=docs/config.example.json go run ./cmd/dbc redis get --profile local-redis --key "user:1"
```

## 配置

默认配置文件路径：

```text
~/.dbconnector/config.json
```

可通过环境变量覆盖：

```bash
DBCONNECTOR_CONFIG=/path/to/config.json
```

配置示例见 [docs/config.example.json](docs/config.example.json)。

完整设计见 [docs/design.md](docs/design.md)。

面向 AI Agent 的通用使用文档见 [docs/agent-usage.md](docs/agent-usage.md)。根目录 [AGENTS.md](AGENTS.md) 也会指向这份文档，方便不同 Agent 自动发现。

## 本机 profile 示例

当前工具会读取：

```text
~/.dbconnector/config.json
```

MySQL 支持两种配置方式。推荐用环境变量保存 DSN：

```bash
export CACTUS_STAGING_MYSQL_DSN='cactus:<PASSWORD>@tcp(proxy-cactus-staging.proxy-ctb6yrxgmjmm.us-west-2.rds.amazonaws.com:3306)/tars_staging?parseTime=true&timeout=5s&readTimeout=10s&writeTimeout=10s'
```

密码中包含 `#` 时，必须用引号包住整段 DSN。

如果需要本机开箱可用，也可以在 `~/.dbconnector/config.json` 的 MySQL profile 中直接配置 `dsn`。`profile list` 只会显示 `dsnConfigured: true`，不会输出 DSN 原文。

连接测试：

```bash
bin/dbc profile test --profile cactus-staging-mysql
bin/dbc profile test --profile cactus-next-redis
```

package app

import (
	"context"
	"io"
	"strings"
	"time"

	"dbconnector/internal/cli"
	"dbconnector/internal/config"
	mysqlclient "dbconnector/internal/mysql"
	"dbconnector/internal/protocol"
	redisclient "dbconnector/internal/redis"
	"dbconnector/internal/safety"
	"dbconnector/internal/version"
)

func Run(args []string, stdout, stderr io.Writer) int {
	start := time.Now()
	if len(args) == 0 {
		return writeFailure(stdout, "system", "", protocol.NewError("USAGE_ERROR", "missing command", false), start)
	}

	switch args[0] {
	case "config":
		return runConfig(args[1:], stdout, start)
	case "profile":
		return runProfile(args[1:], stdout, start)
	case "mysql":
		return runMySQL(args[1:], stdout, start)
	case "redis":
		return runRedis(args[1:], stdout, start)
	case "help", "-h", "--help", "-help":
		return writeSuccess(stdout, "system", "", "help", start, map[string]any{
			"usage":       usage(),
			"commands":    commandList(),
			"commandTree": commandTree(),
		})
	case "version", "-version", "--version":
		info := version.Current()
		return writeSuccess(stdout, "system", "", "version", start, map[string]any{
			"version": info.Version,
			"commit":  info.Commit,
			"builtAt": info.BuiltAt,
		})
	default:
		return writeFailure(stdout, "system", "", protocol.NewError("USAGE_ERROR", "unknown command: "+args[0], false), start)
	}
}

func runConfig(args []string, stdout io.Writer, start time.Time) int {
	if len(args) != 1 || args[0] != "path" {
		return writeFailure(stdout, "system", "", protocol.NewError("USAGE_ERROR", "usage: dbc config path", false), start)
	}

	return writeSuccess(stdout, "system", "", "config_path", start, map[string]any{
		"path": config.DefaultPath(),
		"env":  config.EnvConfigPath,
	})
}

func runProfile(args []string, stdout io.Writer, start time.Time) int {
	if len(args) == 0 {
		return writeFailure(stdout, "system", "", protocol.NewError("USAGE_ERROR", "missing profile command", false), start)
	}

	switch args[0] {
	case "list":
		if len(args) != 1 {
			return writeFailure(stdout, "system", "", protocol.NewError("USAGE_ERROR", "usage: dbc profile list", false), start)
		}
		cfg, errResp := config.Load(config.DefaultPath())
		if errResp != nil {
			return writeFailure(stdout, "system", "", errResp, start)
		}
		return writeSuccess(stdout, "system", "", "profile_list", start, map[string]any{
			"profiles": cfg.SanitizedProfiles(),
			"defaults": cfg.Defaults,
		})
	case "test":
		return runProfileTest(args[1:], stdout, start)
	default:
		return writeFailure(stdout, "system", "", protocol.NewError("USAGE_ERROR", "unknown profile command: "+args[0], false), start)
	}
}

func runProfileTest(args []string, stdout io.Writer, start time.Time) int {
	flags, remaining, errResp := cli.ParseFlags(args)
	if errResp != nil {
		return writeFailure(stdout, "system", "", errResp, start)
	}
	if len(remaining) != 0 {
		return writeFailure(stdout, "system", flags.Profile, protocol.NewError("USAGE_ERROR", "usage: dbc profile test --profile <name>", false), start)
	}
	if strings.TrimSpace(flags.Profile) == "" {
		return writeFailure(stdout, "system", "", protocol.NewError("USAGE_ERROR", "missing required --profile", false), start)
	}

	cfg, errResp := config.Load(config.DefaultPath())
	if errResp != nil {
		return writeFailure(stdout, "system", flags.Profile, errResp, start)
	}

	profile, ok := cfg.FindProfile(flags.Profile)
	if !ok {
		return writeFailure(stdout, "system", flags.Profile, protocol.NewError("PROFILE_NOT_FOUND", "profile not found: "+flags.Profile, false), start)
	}

	engine := strings.ToLower(strings.TrimSpace(profile.Type))
	timeoutMs := effectiveTimeoutMs(cfg, profile, flags.TimeoutMs)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	switch engine {
	case "mysql":
		if errResp := mysqlclient.TestConnection(ctx, *profile); errResp != nil {
			return writeFailure(stdout, engine, profile.Name, errResp, start)
		}
	case "redis":
		if errResp := redisclient.TestConnection(ctx, *profile); errResp != nil {
			return writeFailure(stdout, engine, profile.Name, errResp, start)
		}
	default:
		return writeFailure(stdout, "system", profile.Name, protocol.NewError("PROFILE_TYPE_MISMATCH", "unsupported profile type: "+profile.Type, false), start)
	}

	return writeSuccess(stdout, engine, profile.Name, "connection_test", start, map[string]any{
		"connected": true,
		"timeoutMs": timeoutMs,
	})
}

func runMySQL(args []string, stdout io.Writer, start time.Time) int {
	if len(args) == 0 {
		return writeFailure(stdout, "mysql", "", protocol.NewError("USAGE_ERROR", "missing mysql command", false), start)
	}
	command := args[0]
	flags, cfg, profile, errResp := loadCommandProfile(args[1:], "mysql")
	if errResp != nil {
		return writeFailure(stdout, "mysql", flags.Profile, errResp, start)
	}

	timeoutMs := effectiveTimeoutMs(cfg, profile, flags.TimeoutMs)
	limit := mysqlclient.EffectiveLimit(cfg.Defaults.MaxRows, profile.MaxRows, flags.Limit)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	switch command {
	case "databases":
		databases, errResp := mysqlclient.Databases(ctx, *profile)
		if errResp != nil {
			return writeFailure(stdout, "mysql", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "mysql", profile.Name, "mysql_databases", start, map[string]any{
			"databases": databases,
			"count":     len(databases),
		})
	case "tables":
		tables, truncated, errResp := mysqlclient.Tables(ctx, *profile, flags.Database, limit)
		if errResp != nil {
			return writeFailure(stdout, "mysql", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "mysql", profile.Name, "mysql_tables", start, map[string]any{
			"database":  flags.Database,
			"tables":    tables,
			"count":     len(tables),
			"limit":     limit,
			"truncated": truncated,
		})
	case "table":
		table, errResp := mysqlclient.Table(ctx, *profile, flags.Database, flags.Table)
		if errResp != nil {
			return writeFailure(stdout, "mysql", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "mysql", profile.Name, "mysql_table", start, table)
	case "query":
		if errResp := safety.CheckMySQLRead(flags.SQL); errResp != nil {
			return writeFailure(stdout, "mysql", profile.Name, errResp, start)
		}
		params, errResp := mysqlclient.ParseParams(flags.Params)
		if errResp != nil {
			return writeFailure(stdout, "mysql", profile.Name, errResp, start)
		}
		result, errResp := mysqlclient.Query(ctx, *profile, flags.SQL, params, limit)
		if errResp != nil {
			return writeFailure(stdout, "mysql", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "mysql", profile.Name, "query_result", start, map[string]any{
			"columns":   result.Columns,
			"rows":      result.Rows,
			"rowCount":  result.RowCount,
			"limit":     limit,
			"truncated": result.Truncated,
		})
	case "count":
		if strings.TrimSpace(flags.SQL) != "" && (strings.TrimSpace(flags.Database) != "" || strings.TrimSpace(flags.Table) != "") {
			return writeFailure(stdout, "mysql", profile.Name, protocol.NewError("USAGE_ERROR", "mysql count accepts either --sql or --database with --table, not both", false), start)
		}
		if strings.TrimSpace(flags.SQL) != "" {
			if errResp := safety.CheckMySQLRead(flags.SQL); errResp != nil {
				return writeFailure(stdout, "mysql", profile.Name, errResp, start)
			}
			params, errResp := mysqlclient.ParseParams(flags.Params)
			if errResp != nil {
				return writeFailure(stdout, "mysql", profile.Name, errResp, start)
			}
			result, errResp := mysqlclient.CountQuery(ctx, *profile, flags.SQL, params)
			if errResp != nil {
				return writeFailure(stdout, "mysql", profile.Name, errResp, start)
			}
			return writeSuccess(stdout, "mysql", profile.Name, "mysql_count", start, map[string]any{
				"mode":  "query",
				"count": result.Count,
			})
		}
		result, errResp := mysqlclient.CountTable(ctx, *profile, flags.Database, flags.Table)
		if errResp != nil {
			return writeFailure(stdout, "mysql", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "mysql", profile.Name, "mysql_count", start, map[string]any{
			"mode":     "table",
			"database": flags.Database,
			"table":    flags.Table,
			"count":    result.Count,
		})
	case "explain":
		if strings.TrimSpace(flags.SQL) == "" {
			return writeFailure(stdout, "mysql", profile.Name, protocol.NewError("USAGE_ERROR", "missing required --sql", false), start)
		}
		sqlText := flags.SQL
		if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(sqlText)), "EXPLAIN") {
			sqlText = "EXPLAIN " + sqlText
		}
		if errResp := safety.CheckMySQLRead(sqlText); errResp != nil {
			return writeFailure(stdout, "mysql", profile.Name, errResp, start)
		}
		params, errResp := mysqlclient.ParseParams(flags.Params)
		if errResp != nil {
			return writeFailure(stdout, "mysql", profile.Name, errResp, start)
		}
		result, errResp := mysqlclient.Query(ctx, *profile, sqlText, params, limit)
		if errResp != nil {
			return writeFailure(stdout, "mysql", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "mysql", profile.Name, "explain_result", start, map[string]any{
			"columns":   result.Columns,
			"rows":      result.Rows,
			"rowCount":  result.RowCount,
			"limit":     limit,
			"truncated": result.Truncated,
		})
	case "exec":
		if errResp := safety.CheckWriteAllowed(cfg, profile, flags.Write); errResp != nil {
			return writeFailure(stdout, "mysql", profile.Name, errResp, start)
		}
		if errResp := safety.CheckMySQLWrite(flags.SQL); errResp != nil {
			return writeFailure(stdout, "mysql", profile.Name, errResp, start)
		}
		params, errResp := mysqlclient.ParseParams(flags.Params)
		if errResp != nil {
			return writeFailure(stdout, "mysql", profile.Name, errResp, start)
		}
		result, errResp := mysqlclient.Exec(ctx, *profile, flags.SQL, params)
		if errResp != nil {
			return writeFailure(stdout, "mysql", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "mysql", profile.Name, "exec_result", start, map[string]any{
			"rowsAffected": result.RowsAffected,
			"lastInsertId": result.LastInsertID,
		})
	default:
		return writeFailure(stdout, "mysql", profile.Name, protocol.NewError("USAGE_ERROR", "unknown mysql command: "+command, false), start)
	}
}

func runRedis(args []string, stdout io.Writer, start time.Time) int {
	if len(args) == 0 {
		return writeFailure(stdout, "redis", "", protocol.NewError("USAGE_ERROR", "missing redis command", false), start)
	}
	command := args[0]
	flags, cfg, profile, errResp := loadCommandProfile(args[1:], "redis")
	if errResp != nil {
		return writeFailure(stdout, "redis", flags.Profile, errResp, start)
	}

	timeoutMs := effectiveTimeoutMs(cfg, profile, flags.TimeoutMs)
	limit := mysqlclient.EffectiveLimit(cfg.Defaults.MaxRows, profile.MaxRows, flags.Limit)
	redisProfile := *profile
	if flags.DB != nil {
		redisProfile.DB = flags.DB
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	switch command {
	case "ping":
		pong, errResp := redisclient.Ping(ctx, redisProfile)
		if errResp != nil {
			return writeFailure(stdout, "redis", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "redis", profile.Name, "redis_ping", start, map[string]any{
			"db":   redisDB(redisProfile),
			"pong": pong,
		})
	case "info":
		info, errResp := redisclient.Info(ctx, redisProfile)
		if errResp != nil {
			return writeFailure(stdout, "redis", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "redis", profile.Name, "redis_info", start, map[string]any{
			"db":   redisDB(redisProfile),
			"info": info,
		})
	case "scan":
		keys, truncated, errResp := redisclient.Scan(ctx, redisProfile, flags.Pattern, limit)
		if errResp != nil {
			return writeFailure(stdout, "redis", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "redis", profile.Name, "redis_scan", start, map[string]any{
			"db":        redisDB(redisProfile),
			"pattern":   effectivePattern(flags.Pattern),
			"keys":      keys,
			"count":     len(keys),
			"truncated": truncated,
		})
	case "count":
		count, truncated, errResp := redisclient.Count(ctx, redisProfile, flags.Pattern, flags.Limit)
		if errResp != nil {
			return writeFailure(stdout, "redis", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "redis", profile.Name, "redis_count", start, map[string]any{
			"db":        redisDB(redisProfile),
			"pattern":   effectivePattern(flags.Pattern),
			"count":     count,
			"truncated": truncated,
		})
	case "get":
		value, exists, errResp := redisclient.Get(ctx, redisProfile, flags.Key)
		if errResp != nil {
			return writeFailure(stdout, "redis", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "redis", profile.Name, "redis_get", start, map[string]any{
			"db":     redisDB(redisProfile),
			"key":    flags.Key,
			"exists": exists,
			"value":  value,
		})
	case "hgetall":
		value, errResp := redisclient.HGetAll(ctx, redisProfile, flags.Key)
		if errResp != nil {
			return writeFailure(stdout, "redis", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "redis", profile.Name, "redis_hgetall", start, map[string]any{
			"db":    redisDB(redisProfile),
			"key":   flags.Key,
			"value": value,
			"count": len(value),
		})
	case "ttl":
		ttl, errResp := redisclient.TTL(ctx, redisProfile, flags.Key)
		if errResp != nil {
			return writeFailure(stdout, "redis", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "redis", profile.Name, "redis_ttl", start, map[string]any{
			"db":         redisDB(redisProfile),
			"key":        flags.Key,
			"ttlSeconds": ttl,
		})
	case "type":
		value, errResp := redisclient.Type(ctx, redisProfile, flags.Key)
		if errResp != nil {
			return writeFailure(stdout, "redis", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "redis", profile.Name, "redis_type", start, map[string]any{
			"db":        redisDB(redisProfile),
			"key":       flags.Key,
			"redisType": value,
		})
	case "set":
		if errResp := safety.CheckWriteAllowed(cfg, profile, flags.Write); errResp != nil {
			return writeFailure(stdout, "redis", profile.Name, errResp, start)
		}
		if errResp := redisclient.Set(ctx, redisProfile, flags.Key, flags.Value, flags.TTL); errResp != nil {
			return writeFailure(stdout, "redis", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "redis", profile.Name, "redis_set", start, map[string]any{
			"db":  redisDB(redisProfile),
			"key": flags.Key,
			"ttl": flags.TTL,
		})
	case "del":
		if errResp := safety.CheckWriteAllowed(cfg, profile, flags.Write); errResp != nil {
			return writeFailure(stdout, "redis", profile.Name, errResp, start)
		}
		deleted, errResp := redisclient.Del(ctx, redisProfile, flags.Key)
		if errResp != nil {
			return writeFailure(stdout, "redis", profile.Name, errResp, start)
		}
		return writeSuccess(stdout, "redis", profile.Name, "redis_del", start, map[string]any{
			"db":      redisDB(redisProfile),
			"key":     flags.Key,
			"deleted": deleted,
		})
	default:
		return writeFailure(stdout, "redis", profile.Name, protocol.NewError("USAGE_ERROR", "unknown redis command: "+command, false), start)
	}
}

func redisDB(profile config.Profile) int {
	if profile.DB == nil {
		return 0
	}
	return *profile.DB
}

func loadCommandProfile(args []string, expectedType string) (cli.Flags, *config.Config, *config.Profile, *protocol.Error) {
	flags, remaining, errResp := cli.ParseFlags(args)
	if errResp != nil {
		return flags, nil, nil, errResp
	}
	if len(remaining) != 0 {
		return flags, nil, nil, protocol.NewError("USAGE_ERROR", "unexpected arguments: "+strings.Join(remaining, " "), false)
	}
	if strings.TrimSpace(flags.Profile) == "" {
		return flags, nil, nil, protocol.NewError("USAGE_ERROR", "missing required --profile", false)
	}

	cfg, errResp := config.Load(config.DefaultPath())
	if errResp != nil {
		return flags, nil, nil, errResp
	}
	profile, ok := cfg.FindProfile(flags.Profile)
	if !ok {
		return flags, cfg, nil, protocol.NewError("PROFILE_NOT_FOUND", "profile not found: "+flags.Profile, false)
	}
	if strings.ToLower(strings.TrimSpace(profile.Type)) != expectedType {
		return flags, cfg, profile, protocol.NewError("PROFILE_TYPE_MISMATCH", "profile type mismatch: expected "+expectedType+", got "+profile.Type, false)
	}
	return flags, cfg, profile, nil
}

func effectivePattern(pattern string) string {
	if strings.TrimSpace(pattern) == "" {
		return "*"
	}
	return pattern
}

func effectiveTimeoutMs(cfg *config.Config, profile *config.Profile, override int) int {
	if override > 0 {
		return override
	}
	if profile.TimeoutMs > 0 {
		return profile.TimeoutMs
	}
	return cfg.Defaults.TimeoutMs
}

func writeSuccess(stdout io.Writer, engine, profile, resultType string, start time.Time, fields map[string]any) int {
	resp := protocol.Success(engine, profile, resultType, elapsedMs(start), fields)
	if err := protocol.WriteJSON(stdout, resp); err != nil {
		return 1
	}
	return 0
}

func writeFailure(stdout io.Writer, engine, profile string, errResp *protocol.Error, start time.Time) int {
	resp := protocol.Failure(engine, profile, errResp, elapsedMs(start))
	if err := protocol.WriteJSON(stdout, resp); err != nil {
		return 1
	}
	return 1
}

func elapsedMs(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}

func usage() string {
	return strings.Join([]string{
		"dbc",
		"",
		"Global",
		"  dbc -help",
		"  dbc --help",
		"  dbc -h",
		"  dbc help",
		"  dbc -version",
		"  dbc --version",
		"  dbc version",
		"",
		"Config",
		"  dbc config path",
		"",
		"Profile",
		"  dbc profile list",
		"  dbc profile test --profile <name> [--timeout-ms <ms>]",
		"",
		"MySQL",
		"  dbc mysql databases --profile <name> [--timeout-ms <ms>]",
		"  dbc mysql tables --profile <name> --database <db> [--limit <n>] [--timeout-ms <ms>]",
		"  dbc mysql table --profile <name> --database <db> --table <table> [--timeout-ms <ms>]",
		"  dbc mysql query --profile <name> --sql <select|show|describe|explain> [--params <json-array>] [--limit <n>] [--timeout-ms <ms>]",
		"  dbc mysql count --profile <name> --database <db> --table <table> [--timeout-ms <ms>]",
		"  dbc mysql count --profile <name> --sql <select> [--params <json-array>] [--timeout-ms <ms>]",
		"  dbc mysql explain --profile <name> --sql <select> [--params <json-array>] [--limit <n>] [--timeout-ms <ms>]",
		"  dbc mysql exec --profile <name> --sql <insert|update|delete|...> [--params <json-array>] --write [--timeout-ms <ms>]",
		"",
		"Redis",
		"  dbc redis ping --profile <name> [--db <n>] [--timeout-ms <ms>]",
		"  dbc redis info --profile <name> [--db <n>] [--timeout-ms <ms>]",
		"  dbc redis scan --profile <name> [--db <n>] [--pattern <glob>] [--limit <n>] [--timeout-ms <ms>]",
		"  dbc redis count --profile <name> [--db <n>] [--pattern <glob>] [--limit <n>] [--timeout-ms <ms>]",
		"  dbc redis get --profile <name> [--db <n>] --key <key> [--timeout-ms <ms>]",
		"  dbc redis hgetall --profile <name> [--db <n>] --key <key> [--timeout-ms <ms>]",
		"  dbc redis ttl --profile <name> [--db <n>] --key <key> [--timeout-ms <ms>]",
		"  dbc redis type --profile <name> [--db <n>] --key <key> [--timeout-ms <ms>]",
		"  dbc redis set --profile <name> [--db <n>] --key <key> --value <value> [--ttl <seconds>] --write [--timeout-ms <ms>]",
		"  dbc redis del --profile <name> [--db <n>] --key <key> --write [--timeout-ms <ms>]",
		"",
		"Common Flags",
		"  --profile <name>       Profile name from config",
		"  --timeout-ms <ms>      Per-command timeout override",
		"  --limit <n>            Maximum returned rows, tables, or keys",
		"  --write                Required for write operations",
		"  --params <json-array>  SQL parameter array, e.g. '[123]'",
		"  --db <n>               Redis logical DB override",
	}, "\n")
}

func commandList() []string {
	return []string{
		"config path",
		"profile list",
		"profile test",
		"mysql databases",
		"mysql tables",
		"mysql table",
		"mysql query",
		"mysql count",
		"mysql explain",
		"mysql exec",
		"redis ping",
		"redis info",
		"redis scan",
		"redis count",
		"redis get",
		"redis hgetall",
		"redis ttl",
		"redis type",
		"redis set",
		"redis del",
	}
}

func commandTree() map[string][]string {
	return map[string][]string{
		"global": {
			"dbc -help",
			"dbc --help",
			"dbc -h",
			"dbc help",
			"dbc -version",
			"dbc --version",
			"dbc version",
		},
		"config": {
			"dbc config path",
		},
		"profile": {
			"dbc profile list",
			"dbc profile test --profile <name> [--timeout-ms <ms>]",
		},
		"mysql": {
			"dbc mysql databases --profile <name> [--timeout-ms <ms>]",
			"dbc mysql tables --profile <name> --database <db> [--limit <n>] [--timeout-ms <ms>]",
			"dbc mysql table --profile <name> --database <db> --table <table> [--timeout-ms <ms>]",
			"dbc mysql query --profile <name> --sql <select|show|describe|explain> [--params <json-array>] [--limit <n>] [--timeout-ms <ms>]",
			"dbc mysql count --profile <name> --database <db> --table <table> [--timeout-ms <ms>]",
			"dbc mysql count --profile <name> --sql <select> [--params <json-array>] [--timeout-ms <ms>]",
			"dbc mysql explain --profile <name> --sql <select> [--params <json-array>] [--limit <n>] [--timeout-ms <ms>]",
			"dbc mysql exec --profile <name> --sql <insert|update|delete|...> [--params <json-array>] --write [--timeout-ms <ms>]",
		},
		"redis": {
			"dbc redis ping --profile <name> [--db <n>] [--timeout-ms <ms>]",
			"dbc redis info --profile <name> [--db <n>] [--timeout-ms <ms>]",
			"dbc redis scan --profile <name> [--db <n>] [--pattern <glob>] [--limit <n>] [--timeout-ms <ms>]",
			"dbc redis count --profile <name> [--db <n>] [--pattern <glob>] [--limit <n>] [--timeout-ms <ms>]",
			"dbc redis get --profile <name> [--db <n>] --key <key> [--timeout-ms <ms>]",
			"dbc redis hgetall --profile <name> [--db <n>] --key <key> [--timeout-ms <ms>]",
			"dbc redis ttl --profile <name> [--db <n>] --key <key> [--timeout-ms <ms>]",
			"dbc redis type --profile <name> [--db <n>] --key <key> [--timeout-ms <ms>]",
			"dbc redis set --profile <name> [--db <n>] --key <key> --value <value> [--ttl <seconds>] --write [--timeout-ms <ms>]",
			"dbc redis del --profile <name> [--db <n>] --key <key> --write [--timeout-ms <ms>]",
		},
	}
}

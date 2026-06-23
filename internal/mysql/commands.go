package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"dbconnector/internal/config"
	"dbconnector/internal/protocol"
)

type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type QueryResult struct {
	Columns   []Column         `json:"columns"`
	Rows      []map[string]any `json:"rows"`
	RowCount  int              `json:"rowCount"`
	Truncated bool             `json:"truncated"`
}

type ExecResult struct {
	RowsAffected int64 `json:"rowsAffected"`
	LastInsertID int64 `json:"lastInsertId,omitempty"`
}

func Databases(ctx context.Context, profile config.Profile) ([]string, *protocol.Error) {
	db, errResp := openDB(profile)
	if errResp != nil {
		return nil, errResp
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return nil, driverError(ctx, "mysql databases query failed", err)
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, protocol.NewError("DRIVER_ERROR", "mysql databases scan failed: "+err.Error(), false)
		}
		databases = append(databases, name)
	}
	if err := rows.Err(); err != nil {
		return nil, driverError(ctx, "mysql databases rows failed", err)
	}
	return databases, nil
}

func Tables(ctx context.Context, profile config.Profile, database string, limit int) ([]map[string]any, bool, *protocol.Error) {
	if strings.TrimSpace(database) == "" {
		return nil, false, protocol.NewError("USAGE_ERROR", "missing required --database", false)
	}
	db, errResp := openDB(profile)
	if errResp != nil {
		return nil, false, errResp
	}
	defer db.Close()

	query := `
SELECT table_name, table_type, engine, table_rows
FROM information_schema.tables
WHERE table_schema = ?
ORDER BY table_name`
	result, errResp := QueryRaw(ctx, db, query, []any{database}, limit)
	if errResp != nil {
		return nil, false, errResp
	}
	return result.Rows, result.Truncated, nil
}

func Table(ctx context.Context, profile config.Profile, database, table string) (map[string]any, *protocol.Error) {
	if strings.TrimSpace(database) == "" {
		return nil, protocol.NewError("USAGE_ERROR", "missing required --database", false)
	}
	if strings.TrimSpace(table) == "" {
		return nil, protocol.NewError("USAGE_ERROR", "missing required --table", false)
	}
	db, errResp := openDB(profile)
	if errResp != nil {
		return nil, errResp
	}
	defer db.Close()

	columnsQuery := `
SELECT column_name, column_type, is_nullable, column_key, column_default, extra
FROM information_schema.columns
WHERE table_schema = ? AND table_name = ?
ORDER BY ordinal_position`
	columns, errResp := QueryRaw(ctx, db, columnsQuery, []any{database, table}, 1000)
	if errResp != nil {
		return nil, errResp
	}

	indexesQuery := `
SELECT index_name, non_unique, seq_in_index, column_name
FROM information_schema.statistics
WHERE table_schema = ? AND table_name = ?
ORDER BY index_name, seq_in_index`
	indexes, errResp := QueryRaw(ctx, db, indexesQuery, []any{database, table}, 1000)
	if errResp != nil {
		return nil, errResp
	}

	return map[string]any{
		"database": database,
		"table":    table,
		"columns":  columns.Rows,
		"indexes":  indexes.Rows,
	}, nil
}

func Query(ctx context.Context, profile config.Profile, sqlText string, params []any, limit int) (*QueryResult, *protocol.Error) {
	if strings.TrimSpace(sqlText) == "" {
		return nil, protocol.NewError("USAGE_ERROR", "missing required --sql", false)
	}
	db, errResp := openDB(profile)
	if errResp != nil {
		return nil, errResp
	}
	defer db.Close()
	return QueryRaw(ctx, db, sqlText, params, limit)
}

func QueryRaw(ctx context.Context, db *sql.DB, sqlText string, params []any, limit int) (*QueryResult, *protocol.Error) {
	rows, err := db.QueryContext(ctx, sqlText, params...)
	if err != nil {
		return nil, driverError(ctx, "mysql query failed", err)
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		return nil, protocol.NewError("DRIVER_ERROR", "mysql columns failed: "+err.Error(), false)
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, protocol.NewError("DRIVER_ERROR", "mysql column types failed: "+err.Error(), false)
	}

	columns := make([]Column, 0, len(columnNames))
	for i, name := range columnNames {
		columnType := ""
		if i < len(columnTypes) {
			columnType = columnTypes[i].DatabaseTypeName()
		}
		columns = append(columns, Column{Name: name, Type: columnType})
	}

	resultRows := make([]map[string]any, 0)
	for rows.Next() {
		if limit > 0 && len(resultRows) >= limit {
			return &QueryResult{
				Columns:   columns,
				Rows:      resultRows,
				RowCount:  len(resultRows),
				Truncated: true,
			}, nil
		}

		values := make([]any, len(columnNames))
		dest := make([]any, len(columnNames))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, protocol.NewError("DRIVER_ERROR", "mysql row scan failed: "+err.Error(), false)
		}

		row := make(map[string]any, len(columnNames))
		for i, name := range columnNames {
			row[name] = normalizeValue(values[i])
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, driverError(ctx, "mysql rows failed", err)
	}

	return &QueryResult{
		Columns:   columns,
		Rows:      resultRows,
		RowCount:  len(resultRows),
		Truncated: false,
	}, nil
}

func Exec(ctx context.Context, profile config.Profile, sqlText string, params []any) (*ExecResult, *protocol.Error) {
	if strings.TrimSpace(sqlText) == "" {
		return nil, protocol.NewError("USAGE_ERROR", "missing required --sql", false)
	}
	db, errResp := openDB(profile)
	if errResp != nil {
		return nil, errResp
	}
	defer db.Close()

	result, err := db.ExecContext(ctx, sqlText, params...)
	if err != nil {
		return nil, driverError(ctx, "mysql exec failed", err)
	}
	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()
	return &ExecResult{RowsAffected: rowsAffected, LastInsertID: lastInsertID}, nil
}

func ParseParams(raw string) ([]any, *protocol.Error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var params []any
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		return nil, protocol.NewError("PARAMS_INVALID", "params must be a JSON array: "+err.Error(), false)
	}
	return params, nil
}

func EffectiveLimit(defaultLimit, profileLimit, override int) int {
	if override > 0 {
		return override
	}
	if profileLimit > 0 {
		return profileLimit
	}
	return defaultLimit
}

func openDB(profile config.Profile) (*sql.DB, *protocol.Error) {
	dsn, errResp := mysqlDSN(profile)
	if errResp != nil {
		return nil, errResp
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, protocol.NewError("CONNECT_FAILED", "failed to initialize mysql connection", false)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)
	return db, nil
}

func mysqlDSN(profile config.Profile) (string, *protocol.Error) {
	if dsn := strings.TrimSpace(profile.DSN); dsn != "" {
		return dsn, nil
	}

	envName := strings.TrimSpace(profile.DSNEnv)
	if envName == "" {
		return "", protocol.NewError("CONFIG_INVALID", "mysql profile requires dsn or dsnEnv", false)
	}
	dsn := strings.TrimSpace(os.Getenv(envName))
	if dsn == "" {
		return "", protocol.NewError("ENV_NOT_FOUND", "environment variable not found or empty: "+envName, false)
	}
	return dsn, nil
}

func normalizeValue(value any) any {
	switch v := value.(type) {
	case []byte:
		return string(v)
	default:
		return v
	}
}

func driverError(ctx context.Context, message string, err error) *protocol.Error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return protocol.NewError("TIMEOUT", strings.Replace(message, " failed", " timed out", 1), true)
	}
	return protocol.NewError("DRIVER_ERROR", message+": "+err.Error(), false)
}

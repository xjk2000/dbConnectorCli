package safety

import (
	"strings"
	"unicode"

	"dbconnector/internal/protocol"
)

var mysqlReadKeywords = map[string]bool{
	"SELECT":   true,
	"SHOW":     true,
	"DESCRIBE": true,
	"DESC":     true,
	"EXPLAIN":  true,
}

var mysqlWriteKeywords = map[string]bool{
	"INSERT":   true,
	"UPDATE":   true,
	"DELETE":   true,
	"REPLACE":  true,
	"CREATE":   true,
	"ALTER":    true,
	"DROP":     true,
	"TRUNCATE": true,
}

func CheckMySQLRead(sql string) *protocol.Error {
	keyword, errResp := MySQLFirstKeyword(sql)
	if errResp != nil {
		return errResp
	}
	if HasMySQLMultiStatement(sql) {
		return protocol.NewError("MULTI_STATEMENT_NOT_ALLOWED", "multiple SQL statements are not allowed", false)
	}
	if !mysqlReadKeywords[keyword] {
		return protocol.NewError("SQL_NOT_ALLOWED", "SQL keyword is not allowed for read query: "+keyword, false)
	}
	return nil
}

func CheckMySQLWrite(sql string) *protocol.Error {
	keyword, errResp := MySQLFirstKeyword(sql)
	if errResp != nil {
		return errResp
	}
	if HasMySQLMultiStatement(sql) {
		return protocol.NewError("MULTI_STATEMENT_NOT_ALLOWED", "multiple SQL statements are not allowed", false)
	}
	if !mysqlWriteKeywords[keyword] {
		return protocol.NewError("SQL_NOT_ALLOWED", "SQL keyword is not allowed for exec: "+keyword, false)
	}
	return nil
}

func MySQLFirstKeyword(sql string) (string, *protocol.Error) {
	rest := strings.TrimSpace(stripLeadingComments(sql))
	if rest == "" {
		return "", protocol.NewError("SQL_NOT_ALLOWED", "SQL is empty", false)
	}

	var b strings.Builder
	for _, r := range rest {
		if unicode.IsLetter(r) {
			b.WriteRune(unicode.ToUpper(r))
			continue
		}
		break
	}
	keyword := b.String()
	if keyword == "" {
		return "", protocol.NewError("SQL_NOT_ALLOWED", "SQL must start with a keyword", false)
	}
	return keyword, nil
}

func HasMySQLMultiStatement(sql string) bool {
	inSingle := false
	inDouble := false
	inBacktick := false
	escaped := false

	for i, r := range sql {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && (inSingle || inDouble) {
			escaped = true
			continue
		}
		switch r {
		case '\'':
			if !inDouble && !inBacktick {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle && !inBacktick {
				inDouble = !inDouble
			}
		case '`':
			if !inSingle && !inDouble {
				inBacktick = !inBacktick
			}
		case ';':
			if inSingle || inDouble || inBacktick {
				continue
			}
			if strings.TrimSpace(sql[i+1:]) != "" {
				return true
			}
		}
	}
	return false
}

func stripLeadingComments(sql string) string {
	rest := strings.TrimSpace(sql)
	for {
		switch {
		case strings.HasPrefix(rest, "--"):
			newline := strings.IndexByte(rest, '\n')
			if newline == -1 {
				return ""
			}
			rest = strings.TrimSpace(rest[newline+1:])
		case strings.HasPrefix(rest, "#"):
			newline := strings.IndexByte(rest, '\n')
			if newline == -1 {
				return ""
			}
			rest = strings.TrimSpace(rest[newline+1:])
		case strings.HasPrefix(rest, "/*"):
			end := strings.Index(rest, "*/")
			if end == -1 {
				return ""
			}
			rest = strings.TrimSpace(rest[end+2:])
		default:
			return rest
		}
	}
}

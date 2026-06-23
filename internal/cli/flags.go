package cli

import (
	"strconv"
	"strings"

	"dbconnector/internal/protocol"
)

type Flags struct {
	Profile   string
	TimeoutMs int
	Limit     int
	Write     bool
	Database  string
	Table     string
	SQL       string
	Params    string
	Key       string
	Pattern   string
	Value     string
	Field     string
	TTL       int
	DB        *int
}

func ParseFlags(args []string) (Flags, []string, *protocol.Error) {
	flags := Flags{}
	remaining := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--profile":
			value, ok := nextValue(args, &i, arg)
			if !ok {
				return flags, nil, protocol.NewError("USAGE_ERROR", "missing value for --profile", false)
			}
			flags.Profile = value
		case strings.HasPrefix(arg, "--profile="):
			flags.Profile = strings.TrimPrefix(arg, "--profile=")
		case arg == "--timeout-ms":
			value, ok := nextValue(args, &i, arg)
			if !ok {
				return flags, nil, protocol.NewError("USAGE_ERROR", "missing value for --timeout-ms", false)
			}
			timeoutMs, err := strconv.Atoi(value)
			if err != nil || timeoutMs <= 0 {
				return flags, nil, protocol.NewError("USAGE_ERROR", "--timeout-ms must be a positive integer", false)
			}
			flags.TimeoutMs = timeoutMs
		case strings.HasPrefix(arg, "--timeout-ms="):
			timeoutMs, err := strconv.Atoi(strings.TrimPrefix(arg, "--timeout-ms="))
			if err != nil || timeoutMs <= 0 {
				return flags, nil, protocol.NewError("USAGE_ERROR", "--timeout-ms must be a positive integer", false)
			}
			flags.TimeoutMs = timeoutMs
		case arg == "--limit":
			value, ok := nextValue(args, &i, arg)
			if !ok {
				return flags, nil, protocol.NewError("USAGE_ERROR", "missing value for --limit", false)
			}
			limit, err := strconv.Atoi(value)
			if err != nil || limit <= 0 {
				return flags, nil, protocol.NewError("USAGE_ERROR", "--limit must be a positive integer", false)
			}
			flags.Limit = limit
		case strings.HasPrefix(arg, "--limit="):
			limit, err := strconv.Atoi(strings.TrimPrefix(arg, "--limit="))
			if err != nil || limit <= 0 {
				return flags, nil, protocol.NewError("USAGE_ERROR", "--limit must be a positive integer", false)
			}
			flags.Limit = limit
		case arg == "--write":
			flags.Write = true
		case arg == "--database":
			value, ok := nextValue(args, &i, arg)
			if !ok {
				return flags, nil, protocol.NewError("USAGE_ERROR", "missing value for --database", false)
			}
			flags.Database = value
		case strings.HasPrefix(arg, "--database="):
			flags.Database = strings.TrimPrefix(arg, "--database=")
		case arg == "--table":
			value, ok := nextValue(args, &i, arg)
			if !ok {
				return flags, nil, protocol.NewError("USAGE_ERROR", "missing value for --table", false)
			}
			flags.Table = value
		case strings.HasPrefix(arg, "--table="):
			flags.Table = strings.TrimPrefix(arg, "--table=")
		case arg == "--sql":
			value, ok := nextValue(args, &i, arg)
			if !ok {
				return flags, nil, protocol.NewError("USAGE_ERROR", "missing value for --sql", false)
			}
			flags.SQL = value
		case strings.HasPrefix(arg, "--sql="):
			flags.SQL = strings.TrimPrefix(arg, "--sql=")
		case arg == "--params":
			value, ok := nextValue(args, &i, arg)
			if !ok {
				return flags, nil, protocol.NewError("USAGE_ERROR", "missing value for --params", false)
			}
			flags.Params = value
		case strings.HasPrefix(arg, "--params="):
			flags.Params = strings.TrimPrefix(arg, "--params=")
		case arg == "--key":
			value, ok := nextValue(args, &i, arg)
			if !ok {
				return flags, nil, protocol.NewError("USAGE_ERROR", "missing value for --key", false)
			}
			flags.Key = value
		case strings.HasPrefix(arg, "--key="):
			flags.Key = strings.TrimPrefix(arg, "--key=")
		case arg == "--pattern":
			value, ok := nextValue(args, &i, arg)
			if !ok {
				return flags, nil, protocol.NewError("USAGE_ERROR", "missing value for --pattern", false)
			}
			flags.Pattern = value
		case strings.HasPrefix(arg, "--pattern="):
			flags.Pattern = strings.TrimPrefix(arg, "--pattern=")
		case arg == "--value":
			value, ok := nextValue(args, &i, arg)
			if !ok {
				return flags, nil, protocol.NewError("USAGE_ERROR", "missing value for --value", false)
			}
			flags.Value = value
		case strings.HasPrefix(arg, "--value="):
			flags.Value = strings.TrimPrefix(arg, "--value=")
		case arg == "--field":
			value, ok := nextValue(args, &i, arg)
			if !ok {
				return flags, nil, protocol.NewError("USAGE_ERROR", "missing value for --field", false)
			}
			flags.Field = value
		case strings.HasPrefix(arg, "--field="):
			flags.Field = strings.TrimPrefix(arg, "--field=")
		case arg == "--ttl":
			value, ok := nextValue(args, &i, arg)
			if !ok {
				return flags, nil, protocol.NewError("USAGE_ERROR", "missing value for --ttl", false)
			}
			ttl, err := strconv.Atoi(value)
			if err != nil || ttl < 0 {
				return flags, nil, protocol.NewError("USAGE_ERROR", "--ttl must be a non-negative integer", false)
			}
			flags.TTL = ttl
		case strings.HasPrefix(arg, "--ttl="):
			ttl, err := strconv.Atoi(strings.TrimPrefix(arg, "--ttl="))
			if err != nil || ttl < 0 {
				return flags, nil, protocol.NewError("USAGE_ERROR", "--ttl must be a non-negative integer", false)
			}
			flags.TTL = ttl
		case arg == "--db":
			value, ok := nextValue(args, &i, arg)
			if !ok {
				return flags, nil, protocol.NewError("USAGE_ERROR", "missing value for --db", false)
			}
			db, err := strconv.Atoi(value)
			if err != nil || db < 0 {
				return flags, nil, protocol.NewError("USAGE_ERROR", "--db must be a non-negative integer", false)
			}
			flags.DB = &db
		case strings.HasPrefix(arg, "--db="):
			db, err := strconv.Atoi(strings.TrimPrefix(arg, "--db="))
			if err != nil || db < 0 {
				return flags, nil, protocol.NewError("USAGE_ERROR", "--db must be a non-negative integer", false)
			}
			flags.DB = &db
		default:
			remaining = append(remaining, arg)
		}
	}

	return flags, remaining, nil
}

func nextValue(args []string, index *int, name string) (string, bool) {
	next := *index + 1
	if next >= len(args) || strings.HasPrefix(args[next], "--") {
		return "", false
	}
	*index = next
	return args[next], true
}

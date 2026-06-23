package mysql

import (
	"context"
	"errors"
	"time"

	"dbconnector/internal/config"
	"dbconnector/internal/protocol"

	_ "github.com/go-sql-driver/mysql"
)

func TestConnection(ctx context.Context, profile config.Profile) *protocol.Error {
	db, errResp := openDB(profile)
	if errResp != nil {
		return errResp
	}
	defer db.Close()

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(time.Minute)

	if err := db.PingContext(ctx); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return protocol.NewError("TIMEOUT", "mysql connection test timed out", true)
		}
		return protocol.NewError("CONNECT_FAILED", "mysql connection test failed: "+err.Error(), true)
	}

	return nil
}

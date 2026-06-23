package redis

import (
	"context"
	"errors"

	"dbconnector/internal/config"
	"dbconnector/internal/protocol"
)

func TestConnection(ctx context.Context, profile config.Profile) *protocol.Error {
	client, errResp := newClient(profile)
	if errResp != nil {
		return errResp
	}
	defer client.Close()

	if err := client.Ping(ctx).Err(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return protocol.NewError("TIMEOUT", "redis connection test timed out", true)
		}
		return protocol.NewError("CONNECT_FAILED", "redis connection test failed: "+err.Error(), true)
	}

	return nil
}

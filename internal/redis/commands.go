package redis

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"dbconnector/internal/config"
	"dbconnector/internal/protocol"

	goredis "github.com/redis/go-redis/v9"
)

func Ping(ctx context.Context, profile config.Profile) (string, *protocol.Error) {
	client, errResp := newClient(profile)
	if errResp != nil {
		return "", errResp
	}
	defer client.Close()

	value, err := client.Ping(ctx).Result()
	if err != nil {
		return "", driverError(ctx, "redis ping failed", err)
	}
	return value, nil
}

func Info(ctx context.Context, profile config.Profile) (string, *protocol.Error) {
	client, errResp := newClient(profile)
	if errResp != nil {
		return "", errResp
	}
	defer client.Close()

	value, err := client.Info(ctx).Result()
	if err != nil {
		return "", driverError(ctx, "redis info failed", err)
	}
	return value, nil
}

func Scan(ctx context.Context, profile config.Profile, pattern string, limit int) ([]string, bool, *protocol.Error) {
	client, errResp := newClient(profile)
	if errResp != nil {
		return nil, false, errResp
	}
	defer client.Close()

	if strings.TrimSpace(pattern) == "" {
		pattern = "*"
	}
	count := int64(limit)
	if count <= 0 {
		count = 100
	}

	cursor := uint64(0)
	keys := make([]string, 0)
	for {
		batch, nextCursor, err := client.Scan(ctx, cursor, pattern, count).Result()
		if err != nil {
			return nil, false, driverError(ctx, "redis scan failed", err)
		}
		for _, key := range batch {
			if limit > 0 && len(keys) >= limit {
				return keys, true, nil
			}
			keys = append(keys, key)
		}
		cursor = nextCursor
		if cursor == 0 {
			return keys, false, nil
		}
	}
}

func Get(ctx context.Context, profile config.Profile, key string) (any, bool, *protocol.Error) {
	if strings.TrimSpace(key) == "" {
		return nil, false, protocol.NewError("USAGE_ERROR", "missing required --key", false)
	}
	client, errResp := newClient(profile)
	if errResp != nil {
		return nil, false, errResp
	}
	defer client.Close()

	value, err := client.Get(ctx, key).Result()
	if errors.Is(err, goredis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, driverError(ctx, "redis get failed", err)
	}
	return value, true, nil
}

func HGetAll(ctx context.Context, profile config.Profile, key string) (map[string]string, *protocol.Error) {
	if strings.TrimSpace(key) == "" {
		return nil, protocol.NewError("USAGE_ERROR", "missing required --key", false)
	}
	client, errResp := newClient(profile)
	if errResp != nil {
		return nil, errResp
	}
	defer client.Close()

	value, err := client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, driverError(ctx, "redis hgetall failed", err)
	}
	return value, nil
}

func TTL(ctx context.Context, profile config.Profile, key string) (int64, *protocol.Error) {
	if strings.TrimSpace(key) == "" {
		return 0, protocol.NewError("USAGE_ERROR", "missing required --key", false)
	}
	client, errResp := newClient(profile)
	if errResp != nil {
		return 0, errResp
	}
	defer client.Close()

	value, err := client.TTL(ctx, key).Result()
	if err != nil {
		return 0, driverError(ctx, "redis ttl failed", err)
	}
	return int64(value / time.Second), nil
}

func Type(ctx context.Context, profile config.Profile, key string) (string, *protocol.Error) {
	if strings.TrimSpace(key) == "" {
		return "", protocol.NewError("USAGE_ERROR", "missing required --key", false)
	}
	client, errResp := newClient(profile)
	if errResp != nil {
		return "", errResp
	}
	defer client.Close()

	value, err := client.Type(ctx, key).Result()
	if err != nil {
		return "", driverError(ctx, "redis type failed", err)
	}
	return value, nil
}

func Set(ctx context.Context, profile config.Profile, key, value string, ttlSeconds int) *protocol.Error {
	if strings.TrimSpace(key) == "" {
		return protocol.NewError("USAGE_ERROR", "missing required --key", false)
	}
	client, errResp := newClient(profile)
	if errResp != nil {
		return errResp
	}
	defer client.Close()

	var ttl time.Duration
	if ttlSeconds > 0 {
		ttl = time.Duration(ttlSeconds) * time.Second
	}
	if err := client.Set(ctx, key, value, ttl).Err(); err != nil {
		return driverError(ctx, "redis set failed", err)
	}
	return nil
}

func Del(ctx context.Context, profile config.Profile, key string) (int64, *protocol.Error) {
	if strings.TrimSpace(key) == "" {
		return 0, protocol.NewError("USAGE_ERROR", "missing required --key", false)
	}
	client, errResp := newClient(profile)
	if errResp != nil {
		return 0, errResp
	}
	defer client.Close()

	deleted, err := client.Del(ctx, key).Result()
	if err != nil {
		return 0, driverError(ctx, "redis del failed", err)
	}
	return deleted, nil
}

func newClient(profile config.Profile) (*goredis.Client, *protocol.Error) {
	addr := strings.TrimSpace(profile.Addr)
	if addr == "" {
		return nil, protocol.NewError("CONFIG_INVALID", "redis profile requires addr", false)
	}

	username, errResp := optionalEnv(profile.UsernameEnv)
	if errResp != nil {
		return nil, errResp
	}
	password, errResp := optionalEnv(profile.PasswordEnv)
	if errResp != nil {
		return nil, errResp
	}

	dbIndex := 0
	if profile.DB != nil {
		dbIndex = *profile.DB
	}

	return goredis.NewClient(&goredis.Options{
		Addr:     addr,
		Username: username,
		Password: password,
		DB:       dbIndex,
	}), nil
}

func optionalEnv(name string) (string, *protocol.Error) {
	envName := strings.TrimSpace(name)
	if envName == "" {
		return "", nil
	}

	value := os.Getenv(envName)
	if value == "" {
		return "", protocol.NewError("ENV_NOT_FOUND", "environment variable not found or empty: "+envName, false)
	}
	return value, nil
}

func driverError(ctx context.Context, message string, err error) *protocol.Error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return protocol.NewError("TIMEOUT", strings.Replace(message, " failed", " timed out", 1), true)
	}
	return protocol.NewError("DRIVER_ERROR", message+": "+err.Error(), false)
}

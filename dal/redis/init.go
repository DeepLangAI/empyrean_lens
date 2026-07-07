package redis

import (
	"context"
	"crypto/tls"
	"empyrean_lens/conf"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/redis/go-redis/v9"
)

var redisClient *redis.ClusterClient

func Init() {
	ctx := context.Background()
	config := conf.GetConfig().Redis
	var tlsConfig *tls.Config
	if config.UseTls {
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	redisClient = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      config.Addrs,
		Password:   config.Password,
		Username:   config.Username,
		MaxRetries: 3,

		TLSConfig: tlsConfig,
	})

	err := redisClient.Ping(context.Background()).Err()
	if err != nil {
		panic(err)
	}

	hlog.CtxInfof(ctx, "init redis success")
}

func GetRdb() *redis.ClusterClient {
	return redisClient
}

func KeySet(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return redisClient.Set(ctx, key, value, expiration).Err()
}

// .Val()实际存的值
// .String()执行的命令+值，不要用这个
func GetVal(ctx context.Context, key string) *redis.StringCmd {
	return redisClient.Get(ctx, key)
}
func DelKey(ctx context.Context, key string) error {
	err := redisClient.Del(ctx, key).Err()
	if err != nil {
		hlog.CtxErrorf(ctx, "del key fail, key:%s, err:%s", key, err.Error())
		return err
	}
	return nil
}

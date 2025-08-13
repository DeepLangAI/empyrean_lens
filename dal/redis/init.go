package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"empyrean_lens/conf"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/redis/go-redis/v9"
)

const (
	Stop = 1
)

var (
	rdb       *redis.ClusterClient
	onceRedis sync.Once
)

func Init() {
	onceRedis.Do(func() {
		if rdb == nil {
			rdb = redis.NewClusterClient(conf.GetConfig().Redis)
		}
		if rdb != nil {
			err := rdb.Ping(context.Background()).Err()
			if err != nil {
				logger.Errorf("redis 连接失败. err:%s", err)
				panic("redis 连接失败")
			}
			logger.Info("redis 初始化成功")
		} else {
			panic("redis 连接失败")
		}
	})

	hlog.CtxInfof(context.Background(), "init redis success")
}

func GetRdb() *redis.ClusterClient {
	return rdb
}

func KeySet(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return rdb.Set(ctx, key, value, expiration).Err()
}

func KeySetNx(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	cmd := rdb.SetNX(ctx, key, value, expiration)
	if res, err := cmd.Result(); err != nil || !res {
		return fmt.Errorf("set key fail, key:%s, err:%v", key, err)
	}
	return nil
}

func GetVal(ctx context.Context, key string) *redis.StringCmd {
	return rdb.Get(ctx, key)
}

func DelKey(ctx context.Context, key string) error {
	err := rdb.Del(ctx, key).Err()
	if err != nil {
		hlog.CtxErrorf(ctx, "del key fail, key:%s, err:%s", key, err.Error())
		return err
	}
	return nil
}

func GetStopKey(requestId, sessionId string) string {
	return fmt.Sprintf("stop_answer:%s:%s", requestId, sessionId)
}

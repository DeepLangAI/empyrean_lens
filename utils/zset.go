package utils

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"strconv"
	"time"
)

type ZSet struct {
	rdb     *redis.ClusterClient
	eventNS string // 事件命名空间，例如 "task"
	window  time.Duration
}

func NewZSet(rdb *redis.ClusterClient, eventNS string, window time.Duration) *ZSet {
	return &ZSet{
		rdb:     rdb,
		eventNS: eventNS,
		window:  window,
	}
}
func (z *ZSet) getZSetKey() string {
	return fmt.Sprintf("%s:zset", z.eventNS)
}

func (z *ZSet) Contains(ctx context.Context, md5 string) bool {
	z.flush(ctx)

	key := z.getZSetKey()
	exists := z.rdb.ZScore(ctx, key, md5)
	return exists.Val() > 0
}

func (z *ZSet) Add(ctx context.Context, md5 string) error {
	id := md5
	key := z.getZSetKey()
	now := time.Now().Unix()
	cutoff := now - int64(z.window.Seconds())

	pipe := z.rdb.TxPipeline()
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: id})

	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(cutoff, 10))
	_, err := pipe.Exec(ctx)
	return err
}
func (z *ZSet) flush(ctx context.Context) {
	now := time.Now().Unix()
	cutoff := now - int64(z.window.Seconds())
	key := z.getZSetKey()
	z.rdb.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(cutoff, 10))
}

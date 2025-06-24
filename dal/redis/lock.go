package redis

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type Locker struct {
	ctx context.Context
	key string
	val string
}

func GetLocker(ctx context.Context, key string) *Locker {
	l := &Locker{
		ctx: ctx,
		key: key,
	}
	return l
}

func (l *Locker) getLockKey() string {
	return fmt.Sprintf("lock@%v", l.key)
}

// Unlock 暂未考虑原子性，后续可改用 lua 基本实现 CAD
func (l *Locker) Unlock() error {
	res, err := rdb.Get(l.ctx, l.getLockKey()).Result()
	if err != nil {
		hlog.CtxErrorf(l.ctx, "Unlock Get %v error %v", l.getLockKey(), err)
		return nil
	}
	if res != l.val {
		hlog.CtxWarnf(l.ctx, "ignore unlock, current lock %v, my lock is %v", res, l.val)
		return nil
	}
	if err = rdb.Del(l.ctx, l.getLockKey()).Err(); err != nil {
		hlog.CtxErrorf(l.ctx, "unlock del %v error %v", l.key, err)
		return err
	}
	hlog.CtxInfof(l.ctx, "%v unlock at %v", l.getLockKey(), time.Now())
	return nil
}

func (l *Locker) Lock(lockTime time.Duration) error {
	err := retry.Do(func() error {
		v := strconv.FormatInt(time.Now().UnixMilli(), 10)
		succ, err := rdb.SetNX(l.ctx, l.getLockKey(), v, lockTime).Result()
		if err != nil {
			hlog.CtxErrorf(l.ctx, "GetLock SetNX %v error %v", l.key, err)
			return err
		}
		if !succ {
			time.Sleep(1 * time.Second)
			return fmt.Errorf("%v has been locked", l.key)
		}

		l.val = v
		return nil
	}, retry.Attempts(3))
	if err != nil {
		hlog.CtxErrorf(l.ctx, "lock error: %v", err)
		return err
	}
	hlog.CtxInfof(l.ctx, "%v lock at %v", l.getLockKey(), time.Now())
	return nil
}

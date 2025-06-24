package redis

import (
	"context"
	"empyrean_lens/conf"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	//ctx := context.Background()
	conf.TestInit()
	t.Run("test redis init", func(t *testing.T) {
		Init()
	})
}

func TestKeySet(t *testing.T) {
	ctx := context.Background()
	key := "124"
	conf.TestInit()
	Init()
	t.Run("test redis key set", func(t *testing.T) {
		KeySet(ctx, key, Stop, time.Duration(3)*time.Minute)
		isStop, _ := GetVal(ctx, key).Int()
		println(isStop)
		assert.Equal(t, Stop, isStop)
	})
}

func TestLock(t *testing.T) {
	ctx := context.Background()
	//key := "124"
	conf.TestInit()
	Init()

	t.Run("lock", func(t *testing.T) {
		locker := GetLocker(ctx, "1234:task")
		err := locker.Lock(time.Minute)
		if err != nil {
			hlog.CtxErrorf(ctx, "lock failed: %v", err)
		} else {
			hlog.CtxInfof(ctx, "lock success")
			defer locker.Unlock()
		}
	})
}

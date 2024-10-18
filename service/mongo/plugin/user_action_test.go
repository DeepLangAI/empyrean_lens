package plugin

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"empyrean_lens/utils"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTimeParse(t *testing.T) {
	t1, e := time.Parse("2006-01-02", "1111")
	if e != nil {
		t.Errorf("time parse err:%v", e)
	}
	t.Log(t1.IsZero())
	t1, e = time.Parse("2006-01-02", "2024-11-23")
	if e != nil {
		t.Errorf("time parse err:%v", e)
	}
	t.Log(t1.IsZero())
}

func TestGetUserAction(t *testing.T) {
	uid := "2974a2fbc9f24f0ca65aefc917b98862"
	ctx := context.Background()
	conf.TestInit()
	dal.Init()

	t.Run("uid time", func(t *testing.T) {
		res, e := GetUserAction(ctx, empyrean_lens.GetUserActionReq{Content: "Token", StartTime: "2021-06-20 00:00:00", EndTime: "2024-10-10 00:00:00"})
		if e != nil {
			t.Errorf("get user action err:%v", e)
		} else {
			t.Log(utils.JSONMarshal(res))
		}
	})
	t.Run("uid error", func(t *testing.T) {
		uid = "123"
		res, e := GetUserAction(ctx, empyrean_lens.GetUserActionReq{Content: uid})
		if e != nil {
			t.Errorf("get user action err:%v", e)
		} else {
			t.Log(utils.JSONMarshal(res))
		}
	})
}

func TestSyncMap(t *testing.T) {
	sm := &sync.Map{}
	sm.Store(1, []string{"1", "2", "3"})
	if v, ok := sm.Load(1); ok {
		s := strings.Join(v.([]string), "/")
		t.Log(s)
	}
}

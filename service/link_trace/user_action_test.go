package link_trace

import (
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

func TestSyncMap(t *testing.T) {
	sm := &sync.Map{}
	sm.Store(1, []string{"1", "2", "3"})
	if v, ok := sm.Load(1); ok {
		s := strings.Join(v.([]string), "/")
		t.Log(s)
	}
}

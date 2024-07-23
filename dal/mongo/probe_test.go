package mongo

import (
	"context"
	"empyrean_lens/conf"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
	"time"
)

func TestProbeLogModelDao_Save(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	nodesDetail := []NodeDetail{
		{Name: "login", Cost: 0.8, Result: RESULT_SUCCESS},
		{Name: "logout", Cost: 1.2, Result: RESULT_SUCCESS},
	}
	probeLog := ProbeLogModel{
		Id:           primitive.NewObjectID(),
		TotalNodes:   123,
		SuccessNodes: 10,
		NodesDetail:  nodesDetail,
		Status:       StatusValid,
		CreateTime:   time.Now(),
		UpdateTime:   time.Now(),
	}
	if err := NewProbeLogModelDao().Save(ctx, probeLog); err != nil {
		t.Error(err)
	} else {
		t.Log("success")
	}
}

func TestProbeLogModelDao_FindTimespanProbeLog(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)
	dao := NewProbeLogModelDao()
	logs, err := dao.FindTimespanProbeLog(ctx, time.Now().Add(-time.Hour*24), time.Now())
	if err != nil {
		t.Error(err)
	} else {
		t.Log(logs)
	}
}

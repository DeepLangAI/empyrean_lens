package service

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/dal/mongo"
	"time"
)

func SaveBatch(ctx context.Context, data empyrean_lens.WriteProbeReq) error {
	dao := mongo.NewApiProbeLogModelDao()
	for _, d := range data.Data {
		model := mongo.ApiProbeLogModel{
			Scene:      d.Scene,
			Api:        d.API,
			Host:       d.Host,
			IsCore:     d.IsCore,
			Success:    d.Success,
			Correct:    d.Correct,
			Cost:       d.Cost,
			Status:     0,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
		}
		if err := dao.Save(ctx, model); err != nil {
			return err
		}
	}
	return nil
}

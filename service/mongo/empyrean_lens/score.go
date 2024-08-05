package empyrean_lens

import (
	"context"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"time"
)

func FindScoreDetails(ctx context.Context, timeBegin, timeEnd string) ([]empyrean_lens.SystemScoreModel, error) {
	t1, e := time.Parse("2006-01-02", timeBegin)
	if e != nil {
		return nil, e
	}
	t2, e := time.Parse("2006-01-02", timeEnd)
	if e != nil {
		return nil, e
	}
	dao := empyrean_lens.NewSystemScoreDao()
	score, e := dao.FindTimespanScore(ctx, t1, t2)
	return score, nil
}

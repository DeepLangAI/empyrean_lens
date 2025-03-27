package empyrean_lens

import (
	"context"
	"empyrean_lens/dal/mongo/empyrean_lens"
)

func SaveApiTestDetail(ctx context.Context, model empyrean_lens.ApiTestDetailModel) error {
	return empyrean_lens.NewApiTestDetailDao().Save(ctx, model)
}

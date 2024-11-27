package empyrean_lens

import (
	"context"
	"empyrean_lens/dal/mongo/empyrean_lens"
)

func GetUploadInfos(ctx context.Context) ([]empyrean_lens.UploadLogModel, error) {
	uploadLogDal := empyrean_lens.NewUploadLogModelDao()
	return uploadLogDal.GetUploadInfoByTime(ctx, "")
}

func GetGenerateErrLogInofs(ctx context.Context) ([]empyrean_lens.GenerateErrLogModel, error) {
	generateErrlogDal := empyrean_lens.NewGeneratorErrlogModelDao()
	return generateErrlogDal.GetGenerateErrlogByTime(ctx, "")
}

func GetUploadLogCountByDay(ctx context.Context) (map[string]int64, error) {
	uploadLogDal := empyrean_lens.NewUploadLogModelDao()
	return uploadLogDal.CountLogsByTimeRange(ctx, "2024-11-15")
}

func GetGenerateErrLogCountByDay(ctx context.Context) (map[string]int64, error) {
	generateErrlogDal := empyrean_lens.NewGeneratorErrlogModelDao()
	return generateErrlogDal.CountGenerateErrLogsByTimeRange(ctx, "2024-11-15")
}

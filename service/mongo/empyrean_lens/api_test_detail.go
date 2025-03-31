package empyrean_lens

import (
	"context"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"fmt"
	"time"
)

func SaveApiTestDetail(ctx context.Context, model empyrean_lens.ApiTestDetailModel) error {
	return empyrean_lens.NewApiTestDetailDao().Save(ctx, model)
}

// GetApiTestDetails 获取API测试详情
func GetApiTestDetails(ctx context.Context, apiType string, date time.Time) ([]empyrean_lens.ApiTestDetailModel, error) {
	// 验证 API 类型的合法性
	isValidApiType := false
	for _, validType := range []empyrean_lens.ApiType{
		empyrean_lens.ApiTypeUploadPDF,
		empyrean_lens.ApiTypeUploadPDFParsing,
		empyrean_lens.ApiTypeUploadURL,
		empyrean_lens.ApiTypeEduInput,
		empyrean_lens.ApiTypeEduOutput,
		empyrean_lens.ApiTypeEduTree,
		empyrean_lens.ApiTypeSingleDocOutline,
		empyrean_lens.ApiTypePDFParsing,
		empyrean_lens.ApiTypeTextParse,
		empyrean_lens.ApiTypeWCD,
		empyrean_lens.ApiTypeCrawler,
		empyrean_lens.ApiTypeCrawlerImg,
		empyrean_lens.ApiTypeNovelFormGenerate,
		empyrean_lens.ApiTypeNovelFormGet,
		empyrean_lens.ApiTypeMasterThemeURL,
		empyrean_lens.ApiTypeSingleDocURL,
		empyrean_lens.ApiTypeKeyOpinion,
	} {
		if string(validType) == apiType {
			isValidApiType = true
			break
		}
	}

	if !isValidApiType {
		return nil, fmt.Errorf("invalid api type: %s", apiType)
	}

	return empyrean_lens.NewApiTestDetailDao().FindByApiTypeAndDate(ctx, apiType, date)
}

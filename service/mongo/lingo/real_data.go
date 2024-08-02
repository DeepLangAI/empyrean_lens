package lingo

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/dal/mongo/lingo"
	"time"
)

const (
	ENTRY_TYPE_ABSTRACT  = 5
	ENTRY_TYPE_OUTLINE   = 6
	ENTRY_TYPE_VIEWPOINT = 11
	ENTRY_TYPE_MULTI     = 12
)

const (
	AIGC_TYPE_SUMMAY = 3
)

func RealDataOfDate(ctx context.Context, date string) (*empyrean_lens.RealDataRespData, error) {
	day, err := time.Parse("2006-01-02", date)
	result := &empyrean_lens.RealDataRespData{}
	//result := map[string]int{}
	if err != nil {
		return nil, err
	}

	timeBegin := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.Local)
	timeEnd := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, time.Local)

	dao := lingo.NewWebreaderModelDao()
	webreaderModels, err := dao.FindModels(ctx, timeBegin, timeEnd, true)
	if err != nil {
		return nil, err
	}
	for _, model := range webreaderModels {
		if model.CopyFromUrlId != "" {
			result.PrebuildWeb += 1
		} else {
			result.UploadWeb += 1
		}
		result.Web += 1
	}

	fileModels, err := lingo.NewFileModelDao().FindModels(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}
	for _, model := range fileModels {
		if model.MultiId != "" {
			result.FileInMulti += 1
		} else {
			result.FileInSingle += 1
		}
		result.File += 1
	}

	summaryModels, err := lingo.NewSummaryModelDao().FindModels(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}
	for _, model := range summaryModels {
		if model.EntryType == ENTRY_TYPE_ABSTRACT {
			result.SummaryAbstract += 1
		} else if model.EntryType == ENTRY_TYPE_OUTLINE {
			result.SummaryOutline += 1
		} else if model.EntryType == ENTRY_TYPE_VIEWPOINT {
			result.SummaryViewpoint += 1
		}
	}
	result.Summary = int32(len(summaryModels))

	chatModels, err := lingo.NewChatModelDao().FindModels(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}
	result.Question = int32(len(chatModels))

	chatAnswerModels, err := lingo.NewChatAnswerModelDao().FindModels(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}
	result.Answer = int32(len(chatAnswerModels))

	chatRecommendModels, err := lingo.NewChatRecommendModelDao().FindModels(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}
	result.QuestionRecommend = int32(len(chatRecommendModels))

	multiModels, err := lingo.NewMultiModelDao().FindModels(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}
	result.Multi = int32(len(multiModels))

	multiAigcModels, err := lingo.NewMultiAigcModelDao().FindModels(ctx, timeBegin, timeEnd)
	if err != nil {
		return nil, err
	}
	for _, model := range multiAigcModels {
		if model.AigcType == AIGC_TYPE_SUMMAY {
			result.MultiByTheme += 1
		}
	}

	return result, nil
}

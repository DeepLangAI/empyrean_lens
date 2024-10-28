package lingo

import (
	"context"
	"os"
	"sync"
	"time"

	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/dal/mongo/lingo"

	constslib "codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/consts"
	"github.com/cloudwego/hertz/pkg/common/hlog"
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
	if os.Getenv(constslib.ModeEnvName) == "prod" {
		return result, nil
	}
	//result := map[string]int{}
	if err != nil {
		return nil, err
	}

	timeBegin := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.Local)
	timeEnd := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, time.Local)

	var wg sync.WaitGroup
	var mu sync.Mutex

	wg.Add(1)
	go func() error {
		defer wg.Done()

		hlog.CtxInfof(ctx, "start to get webreader data")
		dao := lingo.NewWebreaderModelDao()
		webreaderModels, err := dao.FindModels(ctx, timeBegin, timeEnd, true)
		if err != nil {
			return err
		}
		hlog.CtxInfof(ctx, "end to get webreader data")
		mu.Lock()
		for _, model := range webreaderModels {
			if model.CopyFromUrlId != "" {
				result.PrebuildWeb += 1
			} else {
				result.UploadWeb += 1
			}
			result.Web += 1
		}
		mu.Unlock()
		return nil
	}()

	wg.Add(1)
	go func() error {
		defer wg.Done()

		hlog.CtxInfof(ctx, "start to get file data")
		fileModels, err := lingo.NewFileModelDao().FindModels(ctx, timeBegin, timeEnd)
		if err != nil {
			return err
		}
		hlog.CtxInfof(ctx, "end to get file data")
		mu.Lock()
		for _, model := range fileModels {
			if model.MultiId != "" {
				result.FileInMulti += 1
			} else {
				result.FileInSingle += 1
			}
			result.File += 1
		}
		mu.Unlock()
		return nil
	}()

	wg.Add(1)
	go func() error {
		defer wg.Done()
		hlog.CtxInfof(ctx, "start to get summary data")
		summaryModels, err := lingo.NewSummaryModelDao().FindModels(ctx, timeBegin, timeEnd)
		if err != nil {
			return err
		}
		mu.Lock()
		hlog.CtxInfof(ctx, "end to get summary data")
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
		mu.Unlock()
		return nil
	}()

	wg.Add(1)
	go func() error {
		defer wg.Done()
		hlog.CtxInfof(ctx, "start to get chat data")
		chatModels, err := lingo.NewChatModelDao().FindModels(ctx, timeBegin, timeEnd)
		if err != nil {
			return err
		}
		hlog.CtxInfof(ctx, "end to get chat data")
		mu.Lock()
		result.Question = int32(len(chatModels))
		mu.Unlock()
		return nil
	}()

	wg.Add(1)
	go func() error {
		defer wg.Done()
		hlog.CtxInfof(ctx, "start to get chat answer data")
		chatAnswerModels, err := lingo.NewChatAnswerModelDao().FindModels(ctx, timeBegin, timeEnd)
		if err != nil {
			return err
		}
		hlog.CtxInfof(ctx, "end to get chat answer data")
		mu.Lock()
		result.Answer = int32(len(chatAnswerModels))
		mu.Unlock()
		return nil
	}()

	wg.Add(1)
	go func() error {
		defer wg.Done()
		hlog.CtxInfof(ctx, "start to get chat recommend data")
		chatRecommendModels, err := lingo.NewChatRecommendModelDao().FindModels(ctx, timeBegin, timeEnd)
		if err != nil {
			return err
		}
		hlog.CtxInfof(ctx, "end to get chat recommend data")
		mu.Lock()
		result.QuestionRecommend = int32(len(chatRecommendModels))
		mu.Unlock()
		return nil
	}()
	wg.Add(1)
	go func() error {
		defer wg.Done()

		hlog.CtxInfof(ctx, "start to get multi data")
		multiModels, err := lingo.NewMultiModelDao().FindModels(ctx, timeBegin, timeEnd)
		if err != nil {
			return err
		}
		hlog.CtxInfof(ctx, "end to get multi data")
		mu.Lock()
		result.Multi = int32(len(multiModels))
		mu.Unlock()
		return nil
	}()

	wg.Add(1)
	go func() error {
		defer wg.Done()
		hlog.CtxInfof(ctx, "start to get multi aigc data")
		multiAigcModels, err := lingo.NewMultiAigcModelDao().FindModels(ctx, timeBegin, timeEnd)
		if err != nil {
			return err
		}
		hlog.CtxInfof(ctx, "end to get multi aigc data")
		for _, model := range multiAigcModels {
			if model.AigcType == AIGC_TYPE_SUMMAY {
				result.MultiByTheme += 1
			}
		}
		return nil
	}()

	wg.Wait()

	return result, nil
}

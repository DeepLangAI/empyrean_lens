package passport

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/utils"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type UserActionInfoTimelineReq struct {
	Uids []string `json:"uids"`
}

func GetUserActionTimeline(ctx context.Context, req empyrean_lens.UserActionInfoReq) ([]*empyrean_lens.UserActionInfoRespData, error) {
	headers := map[string]string{
		"Channel": "local",
	}
	dto := &UserActionInfoTimelineReq{
		Uids: req.Uids,
	}
	resp := empyrean_lens.UserActionInfoResp{}
	err := utils.DoPost(ctx, "https://deeplang-bi.lingoreader.cn/user_action_info", headers, dto, &resp)
	if err != nil {
		hlog.CtxErrorf(ctx, "GetUserActionTimeline error: %v", err)
		return nil, err
	}
	return resp.Data, err
}

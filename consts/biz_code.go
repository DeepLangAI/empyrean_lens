package consts

type BizCode struct {
	// 错误码
	Code int32 `json:"code"`
	// 错误描述
	Msg string `json:"msg"`
}

var (
	ResSuccess         = BizCode{0, "success"}
	RetParamError      = BizCode{100, "参数错误"}
	ParamBindJsonError = BizCode{101, "参数解析错误"}
	SystemErr          = BizCode{102, "服务繁忙，请稍后重试"}

	QueryRecordError = BizCode{201, "数据查询异常"}
	WriteDbError     = BizCode{202, "数据写入异常"}

	LarkAuthError = BizCode{501, "飞书认证失败"}

	// 语鲸原定的错误码
	ID_TOKEN_EXPIRED = BizCode{10010, "login id token failure"}
)

func NewBizErrFromErr(bizErr BizCode, err error) *BizCode {
	return &BizCode{
		Code: bizErr.Code,
		Msg:  bizErr.Msg + ": " + err.Error(),
	}
}

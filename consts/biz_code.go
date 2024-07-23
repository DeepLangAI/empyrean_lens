package consts

type BizCode struct {
	// 错误码
	Code int32
	// 错误描述
	Msg string
}

var (
	ResSuccess         = BizCode{0, "success"}
	RetParamError      = BizCode{100, "参数错误"}
	ParamBindJsonError = BizCode{101, "参数解析错误"}
	SystemErr          = BizCode{102, "服务繁忙，请稍后重试"}

	QueryRecordError = BizCode{201, "数据查询异常"}
	WriteDbError     = BizCode{202, "数据写入异常"}

	// 语鲸原定的错误码
	ID_TOKEN_EXPIRED = BizCode{10010, "login id token failure"}
)

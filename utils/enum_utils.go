package utils

import (
	"empyrean_lens/biz/model/empyrean_lens"
	"fmt"
)

func ChannelIntToString(channel int) string {
	switch empyrean_lens.ChannelType(channel) {
	case empyrean_lens.ChannelType_All:
		return "全部渠道"
	case empyrean_lens.ChannelType_PdfPc,
		empyrean_lens.ChannelType_PdfPlugin,
		empyrean_lens.ChannelType_PdfReader,
		empyrean_lens.ChannelType_PdfPcDb,
		empyrean_lens.ChannelType_PdfWebReader,
		empyrean_lens.ChannelType_UrlPc,
		empyrean_lens.ChannelType_UrlPlugin,
		empyrean_lens.ChannelType_UrlPluginMenu,
		empyrean_lens.ChannelType_UrlPcDb,
		empyrean_lens.ChannelType_UrlReader:
		return "语鲸web"
	case empyrean_lens.ChannelType_WechatUrl,
		empyrean_lens.ChannelType_WechatPdf:
		return "语鲸小助手"
	case empyrean_lens.ChannelType_MiniUrl,
		empyrean_lens.ChannelType_MiniPdf:
		return "语鲸小程序"
	case empyrean_lens.ChannelType_DesktopUrl,
		empyrean_lens.ChannelType_DesktopPdf:
		return "语鲸小程序"
	case empyrean_lens.ChannelType_WebLingoUrl,
		empyrean_lens.ChannelType_WebLingoPdf,
		empyrean_lens.ChannelType_WebLingoMulti:
		return "灵狗web"
	}
	return fmt.Sprintf("%v", channel)
}

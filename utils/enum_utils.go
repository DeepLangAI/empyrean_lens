package utils

import (
	"fmt"
	"strings"
	"time"

	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
)

func ChannelIntToString(channel int) string {
	switch empyrean_lens.ChannelType(channel) {
	case empyrean_lens.ChannelType_All:
		return "订阅"
	case empyrean_lens.ChannelType_PdfPlugin,
		empyrean_lens.ChannelType_UrlPlugin,
		empyrean_lens.ChannelType_UrlPluginMenu:
		return "语鲸插件"
	case empyrean_lens.ChannelType_PdfPc,
		empyrean_lens.ChannelType_PdfReader,
		empyrean_lens.ChannelType_PdfPcDb,
		empyrean_lens.ChannelType_PdfWebReader,
		empyrean_lens.ChannelType_UrlPc,
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
		// 原本是灵狗，但由于前端传参没改回来，只能在这里改成语鲸
		return "语鲸web"
	case empyrean_lens.ChannelType_IosUrl,
		empyrean_lens.ChannelType_IosPdf,
		empyrean_lens.ChannelType_IosMulti,
		empyrean_lens.ChannelType_AndroidUrl,
		empyrean_lens.ChannelType_AndroidPdf,
		empyrean_lens.ChannelType_AndroidMulti,
		empyrean_lens.ChannelType_IosFeedback,
		empyrean_lens.ChannelType_AndroidFeedback:
		return "语鲸app"
	case empyrean_lens.ChannelType_H5Page:
		return "语鲸h5"
	}
	return fmt.Sprintf("%v", channel)
}

func GetActionName(entryType int, multiID string, resourceID string, language string, outlineType int) string {
	// 前缀
	var prefix string
	if resourceID != "" {
		prefix += "订阅-"
	}
	if multiID != "" && entryType != int(empyrean_lens.EntryTypeEnum_MULTI) {
		prefix += "多文档-"
	}
	if language != "" {
		if language == "zh" {
			prefix += "中文-"
		} else {
			prefix += "英文-"
		}
	}
	if entryType == int(empyrean_lens.EntryTypeEnum_MULTI) {
		return prefix + "多文档"
	}
	if entryType == int(empyrean_lens.EntryTypeEnum_WEB) {
		return prefix + "单文档web"
	}
	if entryType == int(empyrean_lens.EntryTypeEnum_FILE) {
		return prefix + "单文档pdf"
	}
	if entryType == int(empyrean_lens.EntryTypeEnum_SUMMARY) {
		return prefix + "概述重新生成"
	}
	if entryType == int(empyrean_lens.EntryTypeEnum_OUTLINE) {
		if outlineType == 2 {
			return prefix + "详细大纲重新生成"
		}
		return prefix + "简单大纲重新生成"
	}
	if entryType == int(empyrean_lens.EntryTypeEnum_VIEWPOINT) {
		return prefix + "关键观点重新生成"
	}
	if entryType == int(empyrean_lens.EntryTypeEnum_MULTI_OUTLINE) {
		return prefix + "大纲重新生成"
	}
	if entryType == int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_WEB) {
		return prefix + "订阅web"
	}
	if entryType == int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_FILE) {
		return prefix + "订阅pdf"
	}
	if entryType == int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_MULTI) {
		return prefix + "订阅多文档"
	}
	return ""
}

func GetStatusFromNode(nodes []*empyrean_lens.GraphNode) (string, empyrean_lens.ActionStatusEnum) {
	// 获取状态，失败原因
	fileAction := ""
	linkStatus := empyrean_lens.ActionStatusEnum_SUCCESS
	for _, node := range nodes {
		if node.Status == empyrean_lens.ActionStatusEnum_FAIL {
			fileAction = node.Name
			linkStatus = empyrean_lens.ActionStatusEnum_FAIL
			break
		}
		if node.Status == empyrean_lens.ActionStatusEnum_WORTHLESS {
			fileAction = node.Name
			linkStatus = empyrean_lens.ActionStatusEnum_WORTHLESS
			break
		}
		if node.Status == empyrean_lens.ActionStatusEnum_LENGTH_ERROR {
			fileAction = node.Name
			linkStatus = empyrean_lens.ActionStatusEnum_LENGTH_ERROR
			break
		}
	}
	if len(nodes) == 1 {
		return fileAction, linkStatus
	}
	// 遍历是否有未执行的
	summaryTypes := []empyrean_lens.LinkNodeTypeEnum{
		empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH,
		empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH,
		empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH,
	}
	for _, node := range nodes {
		if node.Status == empyrean_lens.ActionStatusEnum_UNREACHEAD {
			fileAction = node.Name
			if linkStatus != empyrean_lens.ActionStatusEnum_FAIL && Contains(summaryTypes, node.Type) {
				linkStatus = empyrean_lens.ActionStatusEnum_FAIL
				// 将模型生成未执行的转为失败
				for _, node2 := range nodes {
					if Contains(summaryTypes, node2.Type) && node2.Status == empyrean_lens.ActionStatusEnum_UNREACHEAD {
						node2.Status = empyrean_lens.ActionStatusEnum_FAIL
					}
				}
			}
			break
		}
	}
	return fileAction, linkStatus
}

func GetCostFromNodes(nodes []*empyrean_lens.GraphNode) int {
	if len(nodes) == 0 {
		return 0
	}
	start, end := nodes[0].EnterTime, nodes[0].FinishTime
	for _, node := range nodes {
		if node.FinishTime != "" && strings.Compare(end, node.FinishTime) < 0 {
			end = node.FinishTime
		}
	}
	if start != "" && end != "" {
		startTime, _ := time.Parse(consts.DateTimeTemplate, start)
		endTime, _ := time.Parse(consts.DateTimeTemplate, end)
		return int(endTime.Sub(startTime).Milliseconds())
	}
	return 0
}

func IsSubscribe(entryType int) bool {
	return entryType == int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_WEB) ||
		entryType == int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_FILE) ||
		entryType == int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_MULTI)
}

func IsCopyNodeType(parentEntryType int, nodeType int) bool {
	if !IsSubscribe(parentEntryType) {
		return nodeType == int(empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH) ||
			nodeType == int(empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH) ||
			nodeType == int(empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH) ||
			nodeType == int(empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH) ||
			nodeType == int(empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH) ||
			nodeType == int(empyrean_lens.LinkNodeTypeEnum_MULTI_ANALYSIS_FINISH) ||
			nodeType == int(empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH)
	}
	return nodeType == int(empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH) ||
		nodeType == int(empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH) ||
		nodeType == int(empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH) ||
		nodeType == int(empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH) ||
		nodeType == int(empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH) ||
		nodeType == int(empyrean_lens.LinkNodeTypeEnum_MULTI_ANALYSIS_FINISH) ||
		nodeType == int(empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH) ||
		nodeType == int(empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH) ||
		nodeType == int(empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH) ||
		nodeType == int(empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH) ||
		nodeType == int(empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH)
}

func EntryTypeToNodeType(entryType empyrean_lens.EntryTypeEnum) empyrean_lens.LinkNodeTypeEnum {
	switch entryType {
	case empyrean_lens.EntryTypeEnum_SUMMARY:
		return empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH
	case empyrean_lens.EntryTypeEnum_OUTLINE:
		return empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH
	case empyrean_lens.EntryTypeEnum_VIEWPOINT:
		return empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH
	}
	return 0
}

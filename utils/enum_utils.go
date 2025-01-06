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
		return empyrean_lens.WebSiteLingowhaleSubscribe
	case empyrean_lens.ChannelType_PdfPlugin,
		empyrean_lens.ChannelType_UrlPlugin,
		empyrean_lens.ChannelType_UrlPluginMenu:
		return empyrean_lens.WebSiteLingowhalePlugin
	case empyrean_lens.ChannelType_PdfPc,
		empyrean_lens.ChannelType_PdfReader,
		empyrean_lens.ChannelType_PdfPcDb,
		empyrean_lens.ChannelType_PdfWebReader,
		empyrean_lens.ChannelType_UrlPc,
		empyrean_lens.ChannelType_UrlPcDb,
		empyrean_lens.ChannelType_UrlReader:
		return empyrean_lens.WebSiteLingowhaleWeb
	case empyrean_lens.ChannelType_WechatUrl,
		empyrean_lens.ChannelType_WechatPdf:
		return empyrean_lens.WebSiteLingowhaleHelper
	case empyrean_lens.ChannelType_MiniUrl,
		empyrean_lens.ChannelType_MiniPdf:
		return empyrean_lens.WebSiteLingowhaleMini
	case empyrean_lens.ChannelType_DesktopUrl,
		empyrean_lens.ChannelType_DesktopPdf:
		return empyrean_lens.WebSiteLingowhaleMini
	case empyrean_lens.ChannelType_WebLingoUrl,
		empyrean_lens.ChannelType_WebLingoPdf,
		empyrean_lens.ChannelType_WebLingoMulti:
		// 原本是灵狗，但由于前端传参没改回来，只能在这里改成语鲸
		return empyrean_lens.WebSiteLingowhaleWeb
	case empyrean_lens.ChannelType_IosUrl,
		empyrean_lens.ChannelType_IosPdf,
		empyrean_lens.ChannelType_IosMulti,
		empyrean_lens.ChannelType_IosFeedback:
		return empyrean_lens.WebSiteLingowhaleIos
	case empyrean_lens.ChannelType_AndroidUrl,
		empyrean_lens.ChannelType_AndroidPdf,
		empyrean_lens.ChannelType_AndroidMulti,
		empyrean_lens.ChannelType_AndroidFeedback:
		return empyrean_lens.WebSiteLingowhaleAndroid
	case empyrean_lens.ChannelType_H5Page:
		return empyrean_lens.WebSiteLingowhaleH5
	}
	return fmt.Sprintf("%v", channel)
}

func GetActionName(entryType int, multiID string, resourceID string, language string, outlineType int) (name string) {
	// 前缀
	isMulti := false
	isSubscribe := false
	if multiID != "" && entryType != int(empyrean_lens.EntryTypeEnum_MULTI) {
		isMulti = true
	}
	if resourceID != "" {
		isSubscribe = true
	}
	// web类型
	if entryType == int(empyrean_lens.EntryTypeEnum_WEB) {
		if isMulti {
			return empyrean_lens.ActionNameMultiWeb
		}
		if isSubscribe {
			return empyrean_lens.ActionNameSubscribeCopyWeb
		}
		return empyrean_lens.ActionNameSingleWeb
	}
	// 订阅web类型
	if entryType == int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_WEB) {
		return empyrean_lens.ActionNameSubscribeWeb
	}
	// pdf类型
	if entryType == int(empyrean_lens.EntryTypeEnum_FILE) {
		if isMulti {
			return empyrean_lens.ActionNameMultiPdf
		}
		if isSubscribe {
			return empyrean_lens.ActionNameSubscribeCopyPdf
		}
		return empyrean_lens.ActionNameSinglePdf
	}
	// 订阅pdf类型
	if entryType == int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_FILE) {
		return empyrean_lens.ActionNameSubscribePdf
	}
	// 多文档类型
	if entryType == int(empyrean_lens.EntryTypeEnum_MULTI) {
		if isSubscribe {
			return empyrean_lens.ActionNameSubscribeCopyMulti
		}
		return empyrean_lens.ActionNameMulti
	}
	// 订阅多文档类型
	if entryType == int(empyrean_lens.EntryTypeEnum_SUBSCRIBE_MULTI) {
		return empyrean_lens.ActionNameSubscribeMulti
	}
	// 概述类型
	if entryType == int(empyrean_lens.EntryTypeEnum_SUMMARY) {
		if language == "zh" {
			return empyrean_lens.ActionNameSummaryCH
		}
		if language == "en" {
			return empyrean_lens.ActionNameSummaryEN
		}
		return empyrean_lens.ActionNameSummaryCH
	}
	// 关键观点类型
	if entryType == int(empyrean_lens.EntryTypeEnum_VIEWPOINT) {
		if language == "zh" {
			return empyrean_lens.ActionNameKeyInfoCH
		}
		if language == "en" {
			return empyrean_lens.ActionNameKeyInfoEN
		}
		return empyrean_lens.ActionNameKeyInfoCH
	}
	// 大纲类型
	if entryType == int(empyrean_lens.EntryTypeEnum_OUTLINE) {
		if isMulti {
			return empyrean_lens.ActionNameMultiOutline
		}
		if outlineType == 2 {
			if language == "zh" {
				return empyrean_lens.ActionNameDetailOutlineCH
			}
			if language == "en" {
				return empyrean_lens.ActionNameDetailOutlineEN
			}
			return empyrean_lens.ActionNameDetailOutlineCH
		} else {
			if language == "zh" {
				return empyrean_lens.ActionNameSimpleOutlineCH
			}
			if language == "en" {
				return empyrean_lens.ActionNameSimpleOutlineEN
			}
			return empyrean_lens.ActionNameSimpleOutlineCH
		}
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
		if node.Status == empyrean_lens.ActionStatusEnum_NO_LOG {
			fileAction = node.Name
			linkStatus = empyrean_lens.ActionStatusEnum_NO_LOG
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
		nodeType == int(empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH) ||
		nodeType == int(empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH) ||
		nodeType == int(empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH) ||
		nodeType == int(empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH) ||
		nodeType == int(empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH)
}

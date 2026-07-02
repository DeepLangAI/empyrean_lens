package service

import (
	"fmt"
	"time"

	notice_webhook "empyrean_lens/biz/model/empyrean_lens/notice_webhook"
)

var cstLocation = time.FixedZone("CST", 8*60*60)

type feishuCardMsg struct {
	MsgType string     `json:"msg_type"`
	Card    feishuCard `json:"card"`
}

type feishuCard struct {
	Schema string           `json:"schema"`
	Config feishuCardConfig `json:"config"`
	Header feishuCardHeader `json:"header"`
	Body   feishuCardBody   `json:"body"`
}

type feishuCardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"`
}

type feishuCardHeader struct {
	Title    feishuPlainText `json:"title"`
	Template string          `json:"template"`
}

type feishuPlainText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type feishuCardBody struct {
	Direction string        `json:"direction"`
	Elements  []interface{} `json:"elements"`
}

type feishuMarkdownElement struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type feishuTableElement struct {
	Tag         string                 `json:"tag"`
	PageSize    int                    `json:"page_size"`
	HeaderStyle feishuTableHeaderStyle `json:"header_style"`
	Columns     []feishuTableColumn    `json:"columns"`
	Rows        []map[string]string    `json:"rows"`
}

type feishuTableHeaderStyle struct {
	BackgroundStyle string `json:"background_style"`
}

type feishuTableColumn struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	DataType    string `json:"data_type"`
	Width       string `json:"width"`
}

func buildSlsAlertCard(alert *notice_webhook.SlsAlert, colNames []string) feishuCardMsg {
	template := "red"
	if alert.Status == "resolved" {
		template = "green"
	}

	statusDisplay := "🔥 告警触发"
	if alert.Status == "resolved" {
		statusDisplay = "✅ 告警恢复"
	}

	queryTimeRange := "-"
	if len(alert.Results) > 0 {
		r := alert.Results[0]
		if r.StartTime > 0 && r.EndTime > 0 {
			queryTimeRange = fmt.Sprintf("%s ~ %s", formatUnixTime(r.StartTime), formatUnixTime(r.EndTime))
		}
	}

	details := fmt.Sprintf(
		"**状态**: %s\n**告警时间**: %s\n**触发时间**: %s\n**查询时间范围**: %s\n**项目**: %s\n**区域**: %s\n**告警等级**: %d\n**触发条数**: %d",
		statusDisplay,
		formatUnixTime(alert.AlertTime),
		formatUnixTime(alert.FireTime),
		queryTimeRange,
		alert.Project,
		alert.Region,
		alert.Severity,
		alert.FireResultsCount,
	)

	elements := []interface{}{
		feishuMarkdownElement{Tag: "markdown", Content: details},
	}

	if len(alert.FireResults) > 0 {
		elements = append(elements, buildFireResultsTable(alert.FireResults, colNames))
	}

	if len(alert.Results) > 0 && alert.Results[0].QueryURL != "" {
		link := fmt.Sprintf("[查看查询详情](%s)", alert.Results[0].QueryURL)
		elements = append(elements, feishuMarkdownElement{Tag: "markdown", Content: link})
	}

	return feishuCardMsg{
		MsgType: "interactive",
		Card: feishuCard{
			Schema: "2.0",
			Config: feishuCardConfig{WideScreenMode: true},
			Header: feishuCardHeader{
				Title:    feishuPlainText{Tag: "plain_text", Content: alert.AlertName},
				Template: template,
			},
			Body: feishuCardBody{
				Direction: "vertical",
				Elements:  elements,
			},
		},
	}
}

func buildFireResultsTable(rows []map[string]string, colNames []string) feishuTableElement {
	columns := make([]feishuTableColumn, 0, len(colNames))
	for _, name := range colNames {
		columns = append(columns, feishuTableColumn{
			Name:        name,
			DisplayName: name,
			DataType:    "text",
			Width:       "auto",
		})
	}

	return feishuTableElement{
		Tag:      "table",
		PageSize: 5,
		HeaderStyle: feishuTableHeaderStyle{
			BackgroundStyle: "grey",
		},
		Columns: columns,
		Rows:    rows,
	}
}

func formatUnixTime(ts int64) string {
	if ts == 0 {
		return "-"
	}
	return time.Unix(ts, 0).In(cstLocation).Format("2006-01-02 15:04:05")
}

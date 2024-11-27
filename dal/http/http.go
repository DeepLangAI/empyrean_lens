package http

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/utils"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"strconv"
	"strings"
)

var ShenceDal *shenceDal

type shenceDal struct {
}

type ShenCeResponse struct {
	Code      string `json:"code"`
	RequestID string `json:"request_id"`
	Data      struct {
		Data    []any    `json:"data"`
		Columns []string `json:"columns"`
	} `json:"data"`
}

func (s *shenceDal) executeSql(ctx context.Context, sql string) ([]map[string]string, error) {
	hlog.CtxDebugf(ctx, "execute sql: %s", sql)
	postData := make(map[string]interface{})
	postData["sql"] = sql
	postData["limit"] = "100000"

	res, err := utils.DoPostNew(ctx, conf.GetConfig().ShenCe.Url, postData, conf.GetConfig().ShenCe.Token, conf.GetConfig().ShenCe.Project)
	if err != nil {
		hlog.Errorf("shence execute sql failed: %s", err)
		return nil, err
	}

	entries := strings.Split(res, "\n")

	data := []map[string]string{}
	for _, item := range entries {
		if item == "" {
			continue
		}
		var response ShenCeResponse
		if err := json.Unmarshal([]byte(item), &response); err != nil {
			fmt.Println("Error decoding JSON:", err)
			continue
		}
		respData := response.Data
		columns := respData.Columns

		dataMap := map[string]string{}
		for i, val := range respData.Data {
			key := columns[i]
			dataMap[key] = fmt.Sprintf("%v", val)
		}
		data = append(data, dataMap)
	}
	return data, nil
}

type UploadLogInfo struct {
	UserId        string
	TraceId       string
	FileType      string
	FileName      string
	FileSize      int64
	Ip            string
	FailureReason string
	Date          string
	Time          string
}

func (s *shenceDal) GetUploadInfoByTime(ctx context.Context, dateStr string) ([]UploadLogInfo, error) {

	sqlStr := fmt.Sprintf("SELECT distinct_id,trace_id,failure_reason,file_size,file_name,file_type,$ip,date,time FROM events WHERE event='UploadFile' and product_name='lingowhale' and  date = '%s'", dateStr)

	logs, err := s.executeSql(ctx, sqlStr)
	if err != nil {
		hlog.Errorf("get upload log from shence failed.")
		return nil, err
	}

	UploadInfos := make([]UploadLogInfo, 0)
	for _, log := range logs {
		item := UploadLogInfo{}
		item.UserId = log["distinct_id"]
		item.TraceId = log["trace_id"]
		item.FileType = log["file_type"]
		item.FileName = log["file_name"]
		item.Ip = log["$ip"]

		if log["file_size"] == "" {
			item.FileSize = 0
		} else {
			item.FileSize, _ = strconv.ParseInt(log["file_size"], 10, 64)
		}
		item.FailureReason = log["failure_reason"]
		item.Date = dateStr
		item.Time = log["time"]
		hlog.Info("upload log: %v", item)
		UploadInfos = append(UploadInfos, item)
	}
	return UploadInfos, err
}

type GenerateErrLogInfo struct {
	UserId        string
	TraceId       string
	EntryId       string
	Ip            string
	FailureReason string
	Code          string
	Date          string
	Time          string
}

func (s *shenceDal) GetGenerateErrLogByTime(ctx context.Context, dateStr string) ([]GenerateErrLogInfo, error) {

	sqlStr := fmt.Sprintf("SELECT code,msg,distinct_id,trace_id,entry_type,article_id,$ip,time FROM events WHERE event='Web_Generate_Error' and date = '%s' and code!=0  and product_name='lingowhale' ORDER BY time DESC", dateStr)

	logs, err := s.executeSql(ctx, sqlStr)
	if err != nil {
		hlog.Errorf("get upload log from shence failed.")
		return nil, err
	}

	UploadInfos := make([]GenerateErrLogInfo, 0)
	for _, log := range logs {
		item := GenerateErrLogInfo{}
		item.UserId = log["distinct_id"]
		item.TraceId = log["trace_id"]
		item.EntryId = log["article_id"]
		item.Ip = log["$ip"]
		item.FailureReason = log["msg"]
		item.Code = log["code"]
		item.Date = dateStr
		item.Time = log["time"]
		hlog.Info("upload log: %v", item)
		UploadInfos = append(UploadInfos, item)
	}
	return UploadInfos, err
}

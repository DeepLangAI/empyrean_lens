package link_trace

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	bi "empyrean_lens/dal/mongo/lingowhale_bi"
	"empyrean_lens/utils"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/xuri/excelize/v2"
)

var excelInitOnce sync.Once

type ExcelRow struct {
	Idx        string //序号
	UserID     string //用户ID
	UserType   string //用户类型
	EntryID    string //资源ID
	WebSite    string //产品端
	ActionName string //行为
	Title      string //标题
	UrlList    string //链接列表
	Status     string //状态
	Cost       string //耗时
	CreateAt   string //创建时间
}

func TranslateRespRow(row *empyrean_lens.UserActionRespRow) ExcelRow {
	urlList := []string{}
	for _, resource := range row.Resources {
		urlList = append(urlList, resource.URL)
	}
	return ExcelRow{
		UserID:     row.UserID,
		UserType:   strconv.Itoa(int(row.UserType)),
		EntryID:    row.EntryID,
		WebSite:    row.Channel,
		ActionName: row.ActionName,
		Title:      row.Title,
		UrlList:    strings.Join(urlList, "\n"),
		Status:     row.Status.String(),
		Cost:       strconv.FormatFloat(row.Cost, 'f', 10, 32),
		CreateAt:   row.CreateTime,
	}
}

func MakeUserActionExcel(ctx context.Context, req empyrean_lens.DownloadUserActionReq) (*bytes.Buffer, *consts.BizCode) {
	// 确定时间范围
	var err error
	begin, err := time.ParseInLocation(consts.DateTimeTemplate, req.StartTime, time.Local)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse start time error, err:%v", err)
		return nil, &consts.RetParamError
	}
	end, err := time.ParseInLocation(consts.DateTimeTemplate, req.EndTime, time.Local)
	if err != nil {
		hlog.CtxErrorf(ctx, "parse end time error, err:%v", err)
		return nil, &consts.RetParamError
	}
	if begin.IsZero() && end.IsZero() {
		end = time.Now()
		begin = time.Now().Add(-6 * time.Hour)
	}
	if !begin.Before(end) {
		hlog.CtxErrorf(ctx, "strat time must before end time, start:%v, end:%v", begin, end)
		return nil, &consts.RetParamError
	}
	if end.After(time.Now()) {
		end = time.Now()
	}
	// 从mongo, 读取所有的记录
	getReq := &empyrean_lens.UserActionReq{
		Query:        req.Query,
		Status:       req.Status,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		OnlyExternal: req.OnlyExternal,
		WebSites:     req.WebSites,
		ActionNames:  req.ActionNames,
	}
	totalExcelRows, bizCode := GetExcelRows(ctx, getReq, begin, end)
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "get user action from excel error, err:%v", bizCode)
		return nil, bizCode
	}
	// 保存为 excel
	buffer, err := MakeExccel(ctx, totalExcelRows, "")
	if err != nil {
		hlog.CtxErrorf(ctx, "make excel error, err:%v", err)
		return nil, &consts.RetParamError
	}
	return buffer, nil
}

// 获取数据
func GetExcelRows(ctx context.Context, req *empyrean_lens.UserActionReq, begin, end time.Time) ([]ExcelRow, *consts.BizCode) {
	var excelRows []ExcelRow
	var bizCode *consts.BizCode
	// 有query，直接根据query获取
	// 无query，先从本地excel获取，再从mongo获取
	if req.Query != "" {
		excelRows, bizCode = getGetExcelFromMongo(ctx, req, begin, end)
		if bizCode != nil {
			return nil, bizCode
		}
	} else {
		date := utils.StartDay(begin)
		endDate := utils.StartDay(end)
		// 并发
		wg, lock := sync.WaitGroup{}, sync.Mutex{}
		wg.Add(int(endDate.Sub(date).Hours() / 24))
		for ; date.Before(endDate); date = date.Add(time.Hour * 24) {
			go func(date time.Time) {
				defer wg.Done()
				split, err := getGetExcelFromExcel(ctx, req, date, begin, end)
				if err != nil {
					hlog.CtxErrorf(ctx, "get user action from excel error, err:%v", err)
					bizCode = &consts.RetParamError
					return
				}
				lock.Lock()
				excelRows = append(excelRows, split...)
				lock.Unlock()
			}(date)
		}
		wg.Wait()
	}
	// 如果为空，并且query不为空，查询traceID
	if len(excelRows) == 0 && req.Query != "" {
		rows, bizCode := getUserActionFromTraceID(ctx, req, begin, end)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "get user action from bi error, err:%v", bizCode)
			excelRows = []ExcelRow{}
		} else {
			for _, row := range rows {
				excelRows = append(excelRows, TranslateRespRow(row))
			}
		}
	}
	// 排序
	sort.Slice(excelRows, func(i, j int) bool {
		return excelRows[i].CreateAt < excelRows[j].CreateAt
	})
	// entryID 去重
	idx, unqueTotalRows := 0, []ExcelRow{}
	entryInfoMappinng := map[string]struct{}{}
	for _, row := range excelRows {
		key := fmt.Sprintf("%s_%s", row.ActionName, row.EntryID)
		if _, ok := entryInfoMappinng[key]; !ok {
			entryInfoMappinng[key] = struct{}{}
			row.Idx = strconv.Itoa(idx)
			unqueTotalRows = append(unqueTotalRows, row)
			idx += 1
		}
	}
	return unqueTotalRows, nil
}

// 从mongo获取数据
func getGetExcelFromMongo(ctx context.Context, req *empyrean_lens.UserActionReq, begin, end time.Time) ([]ExcelRow, *consts.BizCode) {
	totalExcelRows := []ExcelRow{}
	offset, limit := int64(0), int64(100)
	for {
		// 直接从bi获取记录
		req.Skip = offset
		req.Limit = limit
		_, _, rows, bizCode := getUserActionFromBi(ctx, req, begin, end, false)
		if bizCode != nil {
			hlog.CtxErrorf(ctx, "get user action from bi error, err:%v", bizCode)
			return nil, bizCode
		}
		for _, row := range rows {
			totalExcelRows = append(totalExcelRows, TranslateRespRow(row))
		}
		// 如果没有下一页，则返回
		if len(rows) < int(limit) {
			break
		}
		offset += limit
	}
	return totalExcelRows, nil
}

// 从excel获取数据
func getGetExcelFromExcel(ctx context.Context, req *empyrean_lens.UserActionReq, date, begin, end time.Time) ([]ExcelRow, *consts.BizCode) {
	// 获取所有数据
	timeAt := date.Format("20060102")
	rows, err := ReadExcel(ctx, fmt.Sprintf("%s.xlsx", timeAt))
	if err != nil || len(rows) == 0 {
		// 从mongo获取当天的数据，记录到本地
		go func() {
			if utils.StartDay(date) != utils.StartDay(time.Now().Local()) {
				RecordExcel(context.Background(), date)
			}
		}()
		// 从mongo读取数据
		hlog.CtxErrorf(ctx, "read excel error, err:%v", err)
		return getGetExcelFromMongo(ctx, req, begin, end)
	}
	// 过滤数据
	totalExcelRows := []ExcelRow{}
	for _, row := range rows {
		// 用户类型过滤
		if req.OnlyExternal && row.UserType != "1" {
			continue
		}
		// 产品端过滤
		if len(req.WebSites) != 0 && !utils.Contains(req.WebSites, row.WebSite) {
			continue
		}
		// 行为过滤
		if len(req.ActionNames) != 0 && !utils.Contains(req.ActionNames, row.ActionName) {
			continue
		}
		// 时间过滤
		start, end := begin.Format(consts.DateTimeTemplate), end.Format(consts.DateTimeTemplate)
		if strings.Compare(start, row.CreateAt) > 0 || strings.Compare(row.CreateAt, end) > 0 {
			continue
		}
		// 状态过滤
		if len(req.Status) != 0 {
			for _, status := range req.Status {
				if status.String() == row.Status {
					totalExcelRows = append(totalExcelRows, row)
					break
				}
			}
		} else {
			totalExcelRows = append(totalExcelRows, row)
		}
	}
	return totalExcelRows, nil
}

// 记录某天的数据到excel
func RecordExcel(ctx context.Context, startTime time.Time) *consts.BizCode {
	start := utils.StartDay(startTime)
	filePath := fmt.Sprintf("../excel/%s.xlsx", start.Format("20060102"))
	// 从mongo获取记录
	buffer, err := bi.NewExcelDao().GridfsDownload(ctx, filePath)
	if err != nil {
		hlog.CtxErrorf(ctx, "download excel from mongo error, err:%v", err)
	}
	if buffer != nil {
		// 写入本地
		err = os.WriteFile(filePath, buffer, 0644)
		if err != nil {
			hlog.CtxErrorf(ctx, "write file error, err:%v", err)
			return &consts.QueryRecordError
		}
		return nil
	}
	// 读取记录
	getReq := &empyrean_lens.UserActionReq{}
	excelRows, bizCode := getGetExcelFromMongo(ctx, getReq, start, start.Add(time.Hour*24))
	if bizCode != nil {
		hlog.CtxErrorf(ctx, "get excel from mongo error, err:%v", bizCode)
		return &consts.QueryRecordError
	}
	// 写入excel
	excel, err := MakeExccel(ctx, excelRows, filePath)
	if err != nil {
		hlog.CtxErrorf(ctx, "make excel error, err:%v", err)
		return &consts.QueryRecordError
	}
	// 保存记录到mongo
	err = bi.NewExcelDao().GridfsUpload(ctx, filePath, excel.Bytes())
	if err != nil {
		hlog.CtxErrorf(ctx, "upload excel from mongo error, err:%v", err)
		return &consts.QueryRecordError
	}
	return nil
}

// 导出
func MakeExccel(ctx context.Context, excelRows []ExcelRow, path string) (*bytes.Buffer, error) {
	// 创建一个excel
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	// 设置首行
	titleList := []string{"序号", "用户ID", "用户类型", "资源ID", "产品端", "行为", "标题", "链接列表", "状态", "耗时", "创建时间"}
	for idx, title := range titleList {
		cellAt := fmt.Sprintf("%s1", string('A'+idx))
		f.SetCellValue("Sheet1", cellAt, title)
	}
	// 设置内容
	for idx, row := range excelRows {
		row.Idx = strconv.Itoa(idx + 1)
		f.SetCellValue("Sheet1", fmt.Sprintf("%s%d", "A", idx+2), row.Idx)
		f.SetCellValue("Sheet1", fmt.Sprintf("%s%d", "B", idx+2), row.UserID)
		f.SetCellValue("Sheet1", fmt.Sprintf("%s%d", "C", idx+2), row.UserType)
		f.SetCellValue("Sheet1", fmt.Sprintf("%s%d", "D", idx+2), row.EntryID)
		f.SetCellValue("Sheet1", fmt.Sprintf("%s%d", "E", idx+2), row.WebSite)
		f.SetCellValue("Sheet1", fmt.Sprintf("%s%d", "F", idx+2), row.ActionName)
		f.SetCellValue("Sheet1", fmt.Sprintf("%s%d", "G", idx+2), row.Title)
		f.SetCellValue("Sheet1", fmt.Sprintf("%s%d", "H", idx+2), row.UrlList)
		f.SetCellValue("Sheet1", fmt.Sprintf("%s%d", "I", idx+2), row.Status)
		f.SetCellValue("Sheet1", fmt.Sprintf("%s%d", "J", idx+2), row.Cost)
		f.SetCellValue("Sheet1", fmt.Sprintf("%s%d", "K", idx+2), row.CreateAt)
	}
	// 返回excel内容
	buffer, err := f.WriteToBuffer()
	if err != nil {
		hlog.CtxErrorf(ctx, "write excel buffer error, err:%v", err)
		return nil, err
	}
	// 保存到本地
	if path != "" {
		err = f.SaveAs(path)
		if err != nil {
			hlog.CtxErrorf(ctx, "save excel error, err:%v", err)
			return nil, err
		}
	}
	return buffer, nil
}

func ReadExcel(ctx context.Context, excelName string) ([]ExcelRow, error) {
	path := fmt.Sprintf("../excel/%s", excelName)
	f, err := excelize.OpenFile(path)
	if err != nil {
		hlog.CtxErrorf(ctx, "read excel error, err:%v", err)
		return nil, err
	}
	// 获取 Sheet1 上所有单元格
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		hlog.CtxErrorf(ctx, "read excel error, err:%v", err)
		return nil, err
	}
	excelRows := []ExcelRow{}
	for idx, row := range rows {
		if idx > 0 {
			excelRows = append(excelRows, ExcelRow{
				Idx:        row[0],
				UserID:     row[1],
				UserType:   row[2],
				EntryID:    row[3],
				WebSite:    row[4],
				ActionName: row[5],
				Title:      row[6],
				UrlList:    row[7],
				Status:     row[8],
				Cost:       row[9],
				CreateAt:   row[10],
			})
		}
	}
	return excelRows, nil
}
